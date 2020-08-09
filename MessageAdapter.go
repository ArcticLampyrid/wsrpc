package wsrpc

// MessageAdapter is an adapter for rpc services to read and write messages.
type MessageAdapter interface {
	// ReadMessage reads a message. If the connection is closed, this function must return an error.
	ReadMessage() ([]byte, error)
	// WriteMessage writes a message. If the connection is closed, this function must return an error.
	WriteMessage(data []byte) error
}
