package wsrpc

import "github.com/gorilla/websocket"

type WebsocketMessageAdapter struct {
	conn *websocket.Conn
}

func NewWebsocketMessageAdapter(conn *websocket.Conn) *WebsocketMessageAdapter {
	return &WebsocketMessageAdapter{conn: conn}
}

func (a *WebsocketMessageAdapter) ReadMessage() ([]byte, error) {
	_, message, err := a.conn.ReadMessage()
	return message, err
}

func (a *WebsocketMessageAdapter) WriteMessage(data []byte) error {
	return a.conn.WriteMessage(websocket.TextMessage, data)
}
