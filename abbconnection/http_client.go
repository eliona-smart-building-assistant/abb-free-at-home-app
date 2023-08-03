package abbconnection

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
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"time"
)

type HttpClient struct {
	Header http.Header
	Client *http.Client
}

func NewHttpClient(useTls bool, checkServerCert bool, connectionTimeoutMs int) *HttpClient {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !checkServerCert},
	}

	httpClient := http.Client{
		Timeout:   time.Duration(connectionTimeoutMs * int(time.Millisecond)),
		Transport: tr,
	}

	c := HttpClient{
		Client: &httpClient,
	}
	return &c
}

func (c *HttpClient) AddHeader(key string, value string) {
	if c.Header == nil {
		c.Header = http.Header{}
	}
	if c.Header.Get(key) == "" {
		c.Header.Add(key, value)
	} else {
		c.Header.Set(key, value)
	}
}

func (c *HttpClient) Request(method string, url string, body *[]byte) ([]byte, int, error) {
	return c.RequestWithQuerys(method, url, body, nil)
}

func (c *HttpClient) RequestWithQuerys(method string, url string, body *[]byte, query *map[string]string) ([]byte, int, error) {
	var responseBody []byte
	var requestHandle *http.Request
	var err error

	if body == nil {
		requestHandle, err = http.NewRequest(method, url, nil)
	} else {
		requestHandle, err = http.NewRequest(method, url, bytes.NewBuffer(*body))
	}

	if err != nil {
		return nil, -1, err
	}

	requestHandle.Header = c.Header

	if query != nil {
		queryHandler := requestHandle.URL.Query()
		for key, value := range *query {
			queryHandler.Add(key, value)
		}
	}

	response, err := c.Client.Do(requestHandle)

	if response != nil && response.Body != nil {
		defer response.Body.Close()
		responseBody, err = ioutil.ReadAll(response.Body)
	}

	if err != nil {
		return nil, -1, err
	}

	return responseBody, response.StatusCode, err
}
