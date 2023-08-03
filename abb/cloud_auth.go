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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	ABB_AUTH_URL        = "https://eu.mybuildings.abb.com/sso/authorize"
	ABB_TOKEN_URL       = "https://eu.mybuildings.abb.com/sso/token"
	ABB_AUTH_CONFIG_URL = "https://api.eu.mybuildings.abb.com/external/oauth2helper/config/"
)

type Oauth2Config struct {
	AuthorizeUrl string `json:"authorize_url"`
	CodeUrl      string `json:"code_url"`
	TokenUrl     string `json:"accesstoken_request_url"`
}

type Oauth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expires      int    `json:"expires_in"`
}

type Oauth2TokenRefreshRequest struct {
	GrantType     string `schema:"grant_type"`
	ClientId      string `schema:"client_id"`
	ClientSecrete string `schema:"client_secret"`
	RefreshToken  string `schema:"refresh_token"`
}

var scopes = []string{
	"RemoteControl",
	"Monitoring",
	"RegisterDevice",
}

type ABBAuth struct {
	oauthConf        *oauth2.Config
	oauth2Code       string
	authorizedClient *http.Client
	oauthToken       *oauth2.Token
	oauthTokenSrc    oauth2.TokenSource
}

func NewABBAuthorization(clientId string, clientSecret string, redirectURL string) *ABBAuth {
	abbAuth := ABBAuth{}

	abbAuth.oauthConf = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   ABB_AUTH_URL,
			TokenURL:  ABB_TOKEN_URL,
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}

	return &abbAuth
}

func randomHex(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func (auth *ABBAuth) Authorize(oauthReturn <-chan OauthReturn) (*string, error) {

	// generate random string for state
	state, err := randomHex(60)
	if err != nil {
		log.Println("couldn't gen random string for oauth state")
	}

	// generate auth url for user to login to abb
	// offline for auto refresh token
	url := auth.oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("\r\n ***********\r\nLogin ONCE with your webbrowser: %v\r\n", url)

	// wait until user called the auth redirect page

	param := <-oauthReturn
	for param.State != state {
		// wrong state
		param = <-oauthReturn
	}

	auth.oauth2Code = param.Code
	log.Println("oauth2 code: ", auth.oauth2Code)

	// get token
	auth.oauthToken, err = auth.oauthConf.Exchange(oauth2.NoContext, auth.oauth2Code, oauth2.AccessTypeOffline)
	if err != nil {
		log.Println("couldn't get token.")
		return nil, err
	}

	log.Println("oauth2 access token: ", auth.oauthToken.AccessToken)

	// http client with token autorefresh ?> auto refresh doesn't work..
	// auth.authorizedClient = auth.oauthConf.Client(oauth2.NoContext, auth.oauthToken)

	auth.oauthTokenSrc = auth.oauthConf.TokenSource(oauth2.NoContext, auth.oauthToken)
	auth.oauthToken, err = auth.oauthTokenSrc.Token()

	if err != nil {
		return nil, err
	}

	return &auth.oauthToken.AccessToken, err
}

func (auth *ABBAuth) Refresh() (*string, error) {
	var err error

	auth.oauthToken, err = auth.oauthTokenSrc.Token()

	if err != nil {
		return nil, err
	}

	return &auth.oauthToken.AccessToken, err
}

func (auth *ABBAuth) GetCurrentAccessToken() string {
	return auth.oauthToken.AccessToken
}

func (auth *ABBAuth) SetCurrentAccessToken(accessToken string) {
	if auth.oauthToken == nil {
		log.Println("WARNING: no token set now. create it.")
		auth.oauthToken = &oauth2.Token{}
	}
	auth.oauthToken.AccessToken = accessToken
}
