package eliona

import (
	"time"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-utils/common"
	"github.com/eliona-smart-building-assistant/go-utils/http"
	"github.com/gorilla/websocket"
)

// on assets (only output attributes). Returns a channel with all changes.
func ListenForOutputChanges() (chan api.Data, error) {
	reconnectTime, _ := time.ParseDuration("30s")
	outputs := make(chan api.Data)
	go http.ListenWebSocketWithReconnect(newWebsocket, reconnectTime, outputs)
	return outputs, nil
}

func newWebsocket() (*websocket.Conn, error) {
	return http.NewWebSocketConnectionWithApiKey(common.Getenv("API_ENDPOINT", "")+"/data-listener?dataSubtype=output", "X-API-Key", common.Getenv("API_TOKEN", ""))
}
