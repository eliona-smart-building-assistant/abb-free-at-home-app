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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

const (
	ABB_AUTH_URL        = "https://eu.mybuildings.abb.com/sso/authorize"
	ABB_TOKEN_URL       = "https://eu.mybuildings.abb.com/sso/token"
	oauth2_redirect_url = "https://api.eu.mybuildings.abb.com/external/oauth2helper/code/set/"
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
	AuthorizedClient *http.Client
	OauthToken       *oauth2.Token
	oauthTokenSrc    oauth2.TokenSource
}

func NewABBAuthorization(clientId string, clientSecret string) *ABBAuth {
	abbAuth := ABBAuth{}

	abbAuth.oauthConf = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  oauth2_redirect_url + clientId,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   ABB_AUTH_URL,
			TokenURL:  ABB_TOKEN_URL,
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}

	return &abbAuth
}

type AuthResponse struct {
	AuthorizeURL          string `json:"authorize_url"`
	CodeURL               string `json:"code_url"`
	AccessTokenRequestURL string `json:"accesstoken_request_url"`
}

type oauthTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
	return t.Transport.RoundTrip(req)
}

func (auth *ABBAuth) AuthorizeOAuth(originalToken *oauth2.Token) (*string, error) {
	auth.OauthToken = originalToken
	if auth.oauthTokenSrc == nil {
		ts := auth.oauthConf.TokenSource(context.Background(), auth.OauthToken)
		auth.oauthTokenSrc = oauth2.ReuseTokenSourceWithExpiry(auth.OauthToken, ts, 2*time.Hour)
	}
	if originalToken == nil {
		resp, err := http.Post("https://api.eu.mybuildings.abb.com/external/oauth2helper/config/"+auth.oauthConf.ClientID, "application/json", bytes.NewBuffer([]byte{}))
		if err != nil {
			return nil, fmt.Errorf("failed to initiate OAuth2 authentication: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-200 response: %s", resp.Status)
		}

		var authResp AuthResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return nil, fmt.Errorf("failed to decode JSON response: %v", err)
		}

		fmt.Printf("\r\n ***********\r\nLogin ONCE with your webbrowser: %v\r\n", authResp.AuthorizeURL)

		// wait until user called the auth redirect page
		code, err := pollForCode(authResp.CodeURL)
		if err != nil {
			return nil, fmt.Errorf("polling for code: %v", err)
		}

		auth.oauth2Code = code

		auth.OauthToken, err = auth.oauthConf.Exchange(context.Background(), auth.oauth2Code, oauth2.AccessTypeOffline)
		if err != nil {
			return nil, fmt.Errorf("getting token from code: %v", err)
		}
		ts := auth.oauthConf.TokenSource(context.Background(), auth.OauthToken)
		auth.oauthTokenSrc = oauth2.ReuseTokenSourceWithExpiry(auth.OauthToken, ts, 2*time.Hour)
	}

	var err error
	auth.OauthToken, err = auth.oauthTokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("getting token from source: %v", err)
	}

	auth.AuthorizedClient = &http.Client{
		Transport: &oauthTransport{
			Token:     auth.OauthToken.AccessToken,
			Transport: http.DefaultTransport,
		},
	}

	// original comment: http client with token autorefresh ?> auto refresh doesn't work..
	// auth.AuthorizedClient = auth.oauthConf.Client(context.Background(), auth.OauthToken)
	return &auth.OauthToken.AccessToken, nil
}

type apiKeyTransport struct {
	Transport http.RoundTripper
	Headers   map[string]string
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range t.Headers {
		req.Header.Add(key, value)
	}
	// Useful for debugging
	// bytes, _ := httputil.DumpRequestOut(req, true)
	// fmt.Printf("%s\n", bytes)
	return t.Transport.RoundTrip(req)
}

func (auth *ABBAuth) AuthorizeAPIKey(key string) {
	auth.AuthorizedClient = &http.Client{
		Transport: &apiKeyTransport{
			Transport: http.DefaultTransport,
			Headers: map[string]string{
				"Authorization": "digest " + key,
			},
		},
	}
}

type CodeResponse struct {
	Code string `json:"code"`
}

func pollForCode(codeURL string) (string, error) {
	maxRetries := 120
	interval := time.Second

	for i := 0; i < maxRetries; i++ {
		code, err := requestForCode(codeURL)
		if err == nil {
			return code, nil
		}

		time.Sleep(interval)
	}

	return "", errors.New("max retries reached without obtaining code")
}

func requestForCode(codeURL string) (string, error) {
	resp, err := http.Get(codeURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var codeResp CodeResponse
		if err := json.NewDecoder(resp.Body).Decode(&codeResp); err != nil {
			return "", err
		}
		return codeResp.Code, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("code not ready yet")
	} else {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
}

func (auth *ABBAuth) Refresh() (*string, error) {
	var err error
	auth.OauthToken, err = auth.oauthTokenSrc.Token()
	if err != nil {
		return nil, err
	}
	return &auth.OauthToken.AccessToken, err
}
