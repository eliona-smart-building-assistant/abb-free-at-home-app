package abb

//  This file is part of the eliona project.
//  Copyright © 2022 LEICOM iTEC AG. All Rights Reserved.
//  ______ _ _
// |  ____| (_)
// | |__  | |_  ___  _ __   __ _
// |  __| | | |/ _ \| '_ \ / _` |
// | |____| | | (_) | | | | (_| |
// |______|_|_|\___/|_| |_|\__,_|
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
//  BUT NOT LIMITED  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//  NON INFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//  DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Original author: Christian Stauffer <christian.stauffer@leicom.ch>

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"abb-free-at-home/abbconnection"
	"abb-free-at-home/abbgraphql"
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"

	"golang.org/x/oauth2"
)

const (
	base_url = "https://api.eu.mybuildings.abb.com"
	// TODO: This shouldn't be hardcoded!
	oauth2_redirect_url    = "https://api.eu.mybuildings.abb.com/external/oauth2helper/code/set/cd1a7768-680d-4040-ab76-b6a6f9c4bf9d"
	API_PATH_CONFIGURATION = "/fhapi/v1/api/rest/configuration"
	API_PATH_UPSTREAM      = "/fhapi/v1/api/rest/datapoint/"
)

type Credentials struct {
	BasicAuth    bool // Local API
	OAuth        bool // MyBuildings Cloud API
	Digest       bool // ProService API key
	User         string
	Password     string
	ClientID     string
	ClientSecret string
	ApiKey       string
	OrgUUID      string
}

type Api struct {
	Credentials Credentials
	Auth        ABBAuth
	BaseUrl     string

	Req *abbconnection.HttpClient

	token *oauth2.Token
}

func NewProServiceApi(config apiserver.Configuration) *Api {
	timeout := int(*config.RequestTimeout)
	api := Api{
		Credentials: Credentials{
			Digest:  true,
			ApiKey:  *config.ApiKey,
			OrgUUID: *config.OrgUUID,
		},
		BaseUrl: base_url,
		Req:     abbconnection.NewHttpClient(true, true, timeout),
	}

	api.Req.AddHeader("Content-Type", "application/json")

	return &api
}

func NewMyBuildingsApi(config apiserver.Configuration) *Api {
	timeout := int(*config.RequestTimeout)
	var token *oauth2.Token
	if config.AccessToken != nil {
		token = &oauth2.Token{
			TokenType:    "bearer",
			AccessToken:  *config.AccessToken,
			RefreshToken: *config.RefreshToken,
			Expiry:       *config.Expiry,
		}
	}
	api := Api{
		Credentials: Credentials{
			OAuth:        true,
			ClientID:     *config.ClientID,
			ClientSecret: *config.ClientSecret,
		},
		Auth:    *NewABBAuthorization(*config.ClientID, *config.ClientSecret, oauth2_redirect_url),
		BaseUrl: base_url,
		Req:     abbconnection.NewHttpClient(true, true, timeout),
		token:   token,
	}

	api.Req.AddHeader("Content-Type", "application/json")

	return &api
}

func NewLocalApi(user string, password string, baseUrl string, timeout int) *Api {
	api := Api{
		Credentials: Credentials{
			BasicAuth: true,
			User:      user,
			Password:  password,
		},
		BaseUrl: baseUrl,
		Req:     abbconnection.NewHttpClient(true, true, timeout),
	}

	api.Req.AddHeader("Content-Type", "application/json")
	api.setAuthHeaders("")
	return &api
}

func (api *Api) Authorize() error {
	switch {
	case api.Credentials.Digest:
		api.Auth.AuthorizeAPIKey(api.Credentials.ApiKey)
		api.setAuthHeaders(api.Credentials.ApiKey)
	case api.Credentials.OAuth:
		accessToken, err := api.Auth.AuthorizeOAuth(api.token)
		if err != nil {
			return fmt.Errorf("obtaining access token: %v", err)
		}
		if accessToken == nil {
			return errors.New("couldn't get authorized client")
		}

		api.setAuthHeaders(*accessToken)
	case api.Credentials.BasicAuth:
		api.setAuthHeaders("")
	}

	return nil
}

func (api *Api) setAuthHeaders(secret string) {
	switch {
	case api.Credentials.Digest:
		api.Req.AddHeader("Authorization", "digest "+secret)
	case api.Credentials.OAuth:
		api.Req.AddHeader("Authorization", "Bearer "+secret)
	case api.Credentials.BasicAuth:
		api.Req.AddHeader("Authorization", "Basic "+
			encodeBase64(api.Credentials.User+":"+api.Credentials.Password))
	}
}

func encodeBase64(plain string) string {
	return base64.StdEncoding.EncodeToString([]byte(plain))
}

func (api *Api) ListenGraphQLSubscriptions(datapoints []appdb.Datapoint, ch chan<- abbgraphql.DataPoint) error {
	if api.Credentials.OAuth {
		return abbgraphql.SubscribeDataPointValue("Bearer "+api.token.AccessToken, datapoints, ch)
	}
	return abbgraphql.SubscribeDataPointValue("digest "+api.Credentials.ApiKey, datapoints, ch)
}

func (api *Api) GetLocations() (abbgraphql.LocationsQuery, error) {
	if api.Auth.AuthorizedClient == nil {
		return abbgraphql.LocationsQuery{}, errors.New("Fetching locations not implemented for legacy API")
	}
	return abbgraphql.GetLocations(api.Auth.AuthorizedClient)
}

func (api *Api) GetConfiguration() (DataFormat, error) {
	if api.Auth.AuthorizedClient == nil {
		return api.getConfigurationLegacy()
	}
	return api.getConfigurationGraphQL()
}

func (api *Api) getConfigurationGraphQL() (DataFormat, error) {
	systemsQueryResult, err := abbgraphql.GetSystems(api.Auth.AuthorizedClient, api.Credentials.OrgUUID)
	if err != nil {
		return DataFormat{}, fmt.Errorf("getting systems from graphQL: %v", err)
	}
	d := convertToDataFormat(systemsQueryResult)
	return d, nil
}

func convertToDataFormat(query abbgraphql.SystemsQuery) DataFormat {
	var dataFormat DataFormat
	dataFormat.Systems = make(map[string]System)
	for _, systemQuery := range query.Systems {
		var system System
		system.SysApName = string(systemQuery.DtId)
		system.Devices = make(map[string]Device)
		for _, asset := range systemQuery.Assets {
			var device Device
			device.DisplayName = string(asset.Name.En)
			device.Location = string(asset.IsLocated.DtId)
			device.Channels = make(map[string]Channel)
			for _, ch := range asset.Channels {
				var channel Channel
				channel.DisplayName = string(ch.Name.En)
				channel.FunctionId = string(ch.FunctionId)
				channel.Outputs = make(map[string]Output)
				channel.Inputs = make(map[string]Input)
				for _, output := range ch.Outputs {
					var out Output
					pairingId, err := strconv.ParseInt(string(output.Value.PairingId), 16, 32)
					if err != nil {
						log.Printf("Error converting pairingId from hex: %v", err)
					}
					out.PairingId = int(pairingId)
					out.Value = string(output.Value.DataPointService.RequestDataPointValue.Value)
					channel.Outputs[string(output.Key)] = out
				}
				for _, input := range ch.Inputs {
					var in Input
					pairingId, err := strconv.ParseInt(string(input.Value.PairingId), 16, 32)
					if err != nil {
						log.Printf("Error converting pairingId from hex: %v", err)
					}
					in.PairingId = int(pairingId)
					in.Value = string(input.Value.DataPointService.RequestDataPointValue.Value)
					channel.Inputs[string(input.Key)] = in
				}
				device.Channels[strconv.Itoa(int(ch.ChannelNumber))] = channel
			}
			system.Devices[string(asset.SerialNumber)] = device
		}
		dataFormat.Systems[string(systemQuery.DtId)] = system
	}

	return dataFormat
}

func (api *Api) getConfigurationLegacy() (DataFormat, error) {
	config := DataFormat{}
	systems := make(map[string]System)

	body, code, err := api.request(abbconnection.REQUEST_METHOD_GET, API_PATH_CONFIGURATION, nil)
	if err != nil {
		return config, fmt.Errorf("requesting configuration API %v: %v", api.BaseUrl+API_PATH_CONFIGURATION, err)
	}
	if code != http.StatusOK {
		return config, fmt.Errorf("configuration %v response with code %d", api.BaseUrl+API_PATH_CONFIGURATION, code)
	}

	err = json.Unmarshal(body, &systems)

	return DataFormat{Systems: systems}, err
}

func (api *Api) WriteDatapoint(system string, deviceId string, channel string, datapoint string, value any) error {
	if api.Auth.AuthorizedClient == nil {
		return api.writeDatapointLegacy(system, deviceId, channel, datapoint, value)
	}
	return api.writeDatapointGraphQL(system, deviceId, channel, datapoint, value)
}

func (api *Api) writeDatapointGraphQL(system string, deviceId string, channel string, datapoint string, value any) error {
	c, err := strconv.Atoi(channel)
	if err != nil {
		return fmt.Errorf("parsing channel number: %v", err)
	}

	return abbgraphql.SetDataPointValue(api.Auth.AuthorizedClient, api.Credentials.Digest, deviceId, c, datapoint, value)
}

func (api *Api) writeDatapointLegacy(system string, deviceId string, channel string, datapoint string, value any) error {
	dpPath := system + "/" + deviceId + "." + channel + "." + datapoint
	reqBody := []byte(fmt.Sprint(value))

	log.Println("write up datapoint:", dpPath, " value: ", string(reqBody))
	body, code, err := api.request(abbconnection.REQUEST_METHOD_PUT, API_PATH_UPSTREAM+dpPath, &reqBody)

	if err != nil {
		return err
	}
	if code != http.StatusOK {
		errorText := fmt.Sprintf("response with code %d", code)
		return errors.New(errorText)
	}

	if !strings.ContainsAny(string(body), "OK") {
		err = errors.New("response is not OK")
	}

	return err
}

func (api *Api) request(method string, path string, payload *[]byte) ([]byte, int, error) {
	var err error
	var accessToken *string

	if !api.Credentials.BasicAuth && !api.Auth.OauthToken.Valid() {
		log.Println("access token expired. renewing")
		accessToken, err = api.Auth.Refresh()
		if err != nil {
			log.Println("error while refreshing token ", err)
			return nil, -1, err
		}
		api.setAuthHeaders(*accessToken)
	}

	return api.Req.Request(method, api.BaseUrl+path, payload)
}
