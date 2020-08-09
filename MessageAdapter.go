package wsrpc

type MessageAdapter interface {
	ReadMessage() ([]byte, error)
	WriteMessage(data []byte) error
}
