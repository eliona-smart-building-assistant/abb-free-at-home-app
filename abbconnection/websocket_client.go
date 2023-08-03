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
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SCHEME_SECURE   = "wss"
	SCHEME_INSECURE = "ws"
)

type WssClient struct {
	CheckCertificate bool
	UseTls           bool
	Url              url.URL
	Connection       *websocket.Conn
	Interrupted      bool
	Header           http.Header
}

func NewWebsocketClient(tls bool, checkCertificate bool) *WssClient {
	websocketClient := WssClient{
		UseTls:           tls,
		CheckCertificate: checkCertificate,
		Interrupted:      false,
	}

	return &websocketClient
}

func (ws *WssClient) AddHeader(key string, value string) {
	if ws.Header == nil {
		ws.Header = http.Header{}
	}
	if ws.Header.Get(key) == "" {
		ws.Header.Add(key, value)
	} else {
		ws.Header.Set(key, value)
	}
}

func (ws *WssClient) CreateConnectionString(uri string) {
	scheme := SCHEME_SECURE
	path := ""
	query := ""

	if !ws.UseTls {
		scheme = SCHEME_INSECURE
	}

	uri = strings.ReplaceAll(uri, "wss://", "")

	host := strings.Split(uri, `/`)[0]
	pathQuery := strings.Split(strings.ReplaceAll(uri, host, ""), `?`)
	if len(pathQuery) >= 1 {
		path = pathQuery[0]
	}
	if len(pathQuery) >= 2 {
		query = pathQuery[1]
	}

	ws.Url = url.URL{Scheme: scheme, Host: host, Path: path, RawQuery: query}
}

func (ws *WssClient) ServeForever(wg *sync.WaitGroup, rxChannel chan<- []byte, interrupt <-chan bool, uri string) {
	defer wg.Done()
	defer close(rxChannel)
	defer log.Println("serve forwever exited")

	var err error
	var response *http.Response

	ws.CreateConnectionString(uri)

	log.Printf("connecting to %s\r\n", ws.Url.String())

	if ws.UseTls {
		tlsConfig := tls.Config{InsecureSkipVerify: ws.CheckCertificate}
		websocket.DefaultDialer.TLSClientConfig = &tlsConfig
	}

	ws.Connection, response, err = websocket.DefaultDialer.Dial(ws.Url.String(), ws.Header)

	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	if err != nil {
		log.Println("wss dial error", err)
		var resp []byte
		if response != nil && response.Body != nil {
			resp, _ = ioutil.ReadAll(response.Body)
		}
		if response != nil {
			log.Printf("handshake failed with response %s", string(resp))
			log.Printf(" | status %d", response.StatusCode)
		}
		return
	}

	if ws.Connection != nil {
		defer ws.Connection.Close()
	}

	readerClosed := make(chan bool)
	go ws.ListenForever(rxChannel, readerClosed)

	for {
		select {
		case <-readerClosed:
			log.Println("Closed Reader")
			return

		case <-interrupt:
			log.Println("Interrupted")
			ws.Interrupted = true
			err := ws.Connection.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("websocket client error while closing: %v\r\n", err)
			}
			// wait for wssReaderLoop
			select {
			case <-readerClosed:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func (ws *WssClient) ListenForever(rxChannel chan<- []byte, closed chan<- bool) {
	// defer close(rxChannel)
	defer close(closed)
	defer log.Println("wss listener forwever exited")

	for {
		_, message, err := ws.Connection.ReadMessage()
		if err != nil {
			log.Printf("websocket client error while listening: %v\r\n", err)
			return
		}
		rxChannel <- message
	}
}

func (ws *WssClient) IsInterrupted() bool {
	return ws.Interrupted
}

func (ws *WssClient) Send(wssConnection *websocket.Conn, message string) {
	err := wssConnection.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("websocket client error while sending: %v\r\n", err)
		return
	}
}
