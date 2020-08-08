package wsrpc

// RPCErrorInfo represents an RPC error that provides error code and message information.
// This type implements error
type RPCErrorInfo struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// RPCInvalidRequestError represents that the JSON sent is not a valid Request object.
var RPCInvalidRequestError = RPCErrorInfo{
	Code:    -32600,
	Message: "Invalid Request"}

// RPCMothedNotFoundError represents an error that method is not found.
var RPCMothedNotFoundError = RPCErrorInfo{
	Code:    -32601,
	Message: "Method not found"}

// RPCInvalidParamsError represents an error that params are invalid.
var RPCInvalidParamsError = RPCErrorInfo{
	Code:    -32602,
	Message: "Invalid params"}

// RPCInternalError represents an internal error.
var RPCInternalError = RPCErrorInfo{
	Code:    -32603,
	Message: "Internal error"}

// RPCParseError represents that an error occurred while parsing the JSON text.
var RPCParseError = RPCErrorInfo{
	Code:    -32700,
	Message: "Parse error"}

func (err RPCErrorInfo) Error() string {
	return err.Message
}
