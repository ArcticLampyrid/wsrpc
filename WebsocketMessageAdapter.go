package wsrpc

import "github.com/gorilla/websocket"

// WebsocketMessageAdapter is an adapter for rpc services to work with gorilla/websocket
type WebsocketMessageAdapter struct {
	conn *websocket.Conn
}

// NewWebsocketMessageAdapter creates a adapter.
func NewWebsocketMessageAdapter(conn *websocket.Conn) *WebsocketMessageAdapter {
	return &WebsocketMessageAdapter{conn: conn}
}

// ReadMessage reads a message. If the connection is closed, this function must return an error.
func (a *WebsocketMessageAdapter) ReadMessage() ([]byte, error) {
	_, message, err := a.conn.ReadMessage()
	return message, err
}

// WriteMessage writes a message. If the connection is closed, this function must return an error.
func (a *WebsocketMessageAdapter) WriteMessage(data []byte) error {
	return a.conn.WriteMessage(websocket.TextMessage, data)
}
