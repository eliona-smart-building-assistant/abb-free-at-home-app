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
	"sync"
	"time"

	"abb-free-at-home/abbconnection"
	"abb-free-at-home/abbgraphql"
	"abb-free-at-home/apiserver"

	utilslog "github.com/eliona-smart-building-assistant/go-utils/log"
	"golang.org/x/oauth2"
)

const (
	API_PATH_WS_ADDR        = "/fhapi/v1/api/ws"
	API_PATH_CONFIGURATION  = "/fhapi/v1/api/rest/configuration"
	API_PATH_UPSTREAM       = "/fhapi/v1/api/rest/datapoint/"
	WEBSOCK_RAW_BUFFER_SIZE = 1000
)

type Credentials struct {
	BasicAuth              bool
	User                   string
	Password               string
	ClientID               string
	ClientSecret           string
	OcpApimSubscriptionKey string
}

type Api struct {
	Credentials Credentials
	Auth        ABBAuth
	BaseUrl     string

	Req       *abbconnection.HttpClient
	Websocket abbconnection.WebSocketInterface
	wssUrl    string

	WebsocketUp bool
	Timeout     int

	token *oauth2.Token

	tokenCheckTicker *time.Ticker
}

func NewGraphQLApi(config apiserver.Configuration, baseUrl string, oauth2RedirectURL string) *Api {
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
			BasicAuth:    false,
			ClientID:     *config.ClientID,
			ClientSecret: *config.ClientSecret,
		},
		Auth:        *NewABBAuthorization(*config.ClientID, *config.ClientSecret, oauth2RedirectURL),
		BaseUrl:     baseUrl,
		Req:         abbconnection.NewHttpClient(true, true, timeout),
		WebsocketUp: false,
		Timeout:     timeout,
		token:       token,
	}

	api.Req.AddHeader("Content-Type", "application/json")

	return &api
}

func NewLocalApi(user string, password string,
	baseUrl string, timeout int) *Api {
	api := Api{
		Credentials: Credentials{
			BasicAuth: true,
			User:      user,
			Password:  password,
		},
		BaseUrl:     baseUrl,
		Req:         abbconnection.NewHttpClient(true, true, timeout),
		WebsocketUp: false,
		Timeout:     timeout,
	}

	api.Req.AddHeader("Content-Type", "application/json")
	api.setAuthHeaders(nil)
	return &api
}

var tokenCheckerOnce sync.Once

func (api *Api) Authorize() error {
	if !api.Credentials.BasicAuth {
		// cloud instance with oauth2
		accessToken, err := api.Auth.Authorize(api.token)
		if err != nil {
			return fmt.Errorf("obtaining access token: %v", err)
		}
		if accessToken == nil {
			return errors.New("couldn't get authorized client")
		}

		if err := api.setAuthHeaders(accessToken); err != nil {
			return fmt.Errorf("setting auth headers: %v", err)
		}
		go func() {
			tokenCheckerOnce.Do(api.tokenChecker)
		}()
	} else {
		// local instaces uses
		if err := api.setAuthHeaders(nil); err != nil {
			return fmt.Errorf("setting auth headers: %v", err)
		}
	}

	return nil
}

func (api *Api) tokenChecker() {
	api.tokenCheckTicker = time.NewTicker(1 * time.Second)
	fmt.Println("start token validator")
	for {
		select {
		case _, ok := <-api.tokenCheckTicker.C:
			if ok {
				if !api.Auth.OauthToken.Valid() {
					fmt.Println("reauthorizing token")
					// todo: make something to autorenew token
					if err := api.Authorize(); err != nil {
						utilslog.Error("abb", "reauthorizing token: %v", err)
					}
				}
			} else {
				log.Println("ticker exited")
				return
			}
		}
	}
}

func (api *Api) setAuthHeaders(accessToken *string) error {
	var err error
	if !api.Credentials.BasicAuth {
		if accessToken != nil {
			api.Req.AddHeader("Ocp-Apim-Subscription-Key", api.Credentials.OcpApimSubscriptionKey)
			api.Req.AddHeader("Authorization", "Bearer "+*accessToken)
		} else {
			err = errors.New("no access token given to set")
		}
	} else {
		api.Req.AddHeader("Authorization", "Basic "+
			encodeBase64(api.Credentials.User+":"+api.Credentials.Password))
	}
	return err
}

func encodeBase64(plain string) string {
	return base64.StdEncoding.EncodeToString([]byte(plain))
}

func (api *Api) UpdateBearerManually(jwt string) {
	api.Auth.SetCurrentAccessToken(jwt)

	api.Req.AddHeader("Ocp-Apim-Subscription-Key", api.Credentials.OcpApimSubscriptionKey)
	api.Req.AddHeader("Authorization", "Bearer "+api.Auth.GetCurrentAccessToken())
}

// ToDo: for local instances available?
func (api *Api) ListenOnEvents(wg *sync.WaitGroup, events chan<- WsObject, ir <-chan bool) {
	defer wg.Done()
	defer close(events)
	defer log.Println("abb event listener exited")

	var wgWs sync.WaitGroup

	wssUrl, err := api.GetWebsocketUrl()

	if err != nil {
		log.Println("error while getting websocket address: ", err)
		wssUrl = "wss://fhapi.my.busch-jaeger.de/api/ws"
	}

	api.wssUrl = wssUrl

	interrupted := false
	for !interrupted {
		api.Websocket = abbconnection.NewWebsocketClient(true, true)
		if api.Websocket == nil {
			log.Println("couldn't get a websocket")
			return
		}
		// todo: make loop while not interrupted and get "new" token
		api.Websocket.AddHeader("Ocp-Apim-Subscription-Key", api.Credentials.OcpApimSubscriptionKey)
		api.Websocket.AddHeader("Authorization", "Bearer "+api.Auth.GetCurrentAccessToken())

		log.Println("wss: sub key ", api.Credentials.OcpApimSubscriptionKey)
		log.Println("wss: token ", api.Auth.GetCurrentAccessToken())
		wgWs.Add(1)

		rxRaw := make(chan []byte, WEBSOCK_RAW_BUFFER_SIZE)
		irWs := make(chan bool)

		go api.Websocket.ServeForever(&wgWs, rxRaw, irWs, api.wssUrl)

		api.WebsocketUp = true
		serve := true
		for serve {
			select {
			case raw, ok := <-rxRaw:
				if !ok {
					log.Println("raw wss channel closed by wss client")
					// irWs <- true
					serve = false
					break
				}

				var inJson WsObject
				err := json.Unmarshal(raw, &inJson)
				if err != nil {
					log.Println("invalid data received by ws: ", err, rxRaw)
				} else {
					events <- inJson
				}

			case irupt, ok := <-ir:
				if !ok || irupt {
					log.Println("interrupted")
					// non blocking?
					interrupted = irupt
					irWs <- irupt
					goto exit
				}
			}
		}

		log.Println("try to restarting wss")
		// irWs <- false
		wgWs.Wait()
		time.Sleep(1 * time.Second)
		log.Println("restart wss")
	}

exit:

	if api.tokenCheckTicker != nil {
		api.tokenCheckTicker.Stop()
	}

	log.Println("wait for websocket serve loop")
	wgWs.Wait()
	api.WebsocketUp = false
}

func (api *Api) GetWebsocketUrl() (string, error) {
	var url string

	body, code, err := api.request(abbconnection.REQUEST_METHOD_GET, API_PATH_WS_ADDR, nil)
	if err != nil {
		errTxt := err.Error()

		if strings.Contains(errTxt, "unsupported protocol scheme \"wss\"") {
			start := strings.Index(errTxt, "wss://")
			end := strings.LastIndex(errTxt, ":")
			if start != -1 && end != -1 {
				url = errTxt[start:end]
			}
		}

		if len(url) > 0 {
			err = nil
		}
	} else {
		log.Println("**not implemented** error: while getting websocket url", code, string(body))
		err = errors.New("error get websocket url")
	}

	// ToDo: check, why %22 is there ***************************************3
	strings.ReplaceAll(url, "%22", "")
	fmt.Println("* ws address (todo): ", url)
	url = "wss://fhapi.my.busch-jaeger.de/api/ws"

	return url, err
}

func (api *Api) GetConfiguration() (DataFormat, error) {
	if api.Auth.AuthorizedClient == nil {
		return api.getConfigurationLegacy()
	}
	return api.getConfigurationGraphQL()
}

func (api *Api) getConfigurationGraphQL() (DataFormat, error) {
	systemsQueryResult, err := abbgraphql.GetSystems(api.Auth.AuthorizedClient)
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
			device.Channels = make(map[string]Channel)

			for _, ch := range asset.Channels {
				var channel Channel
				channel.DisplayName = string(ch.Name.En)
				channel.FunctionId = string(ch.FunctionId)
				channel.Outputs = make(map[string]Output)

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

func (api *Api) WriteDatapoint(system string, deviceId string, channel string, datapoint string, value interface{}) error {
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
		err = api.setAuthHeaders(accessToken)
		if err != nil {
			log.Println("error while renewing access token: ", err)
			return nil, -1, err
		}
	}

	return api.Req.Request(method, api.BaseUrl+path, payload)
}
