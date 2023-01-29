package wsrpc

import (
	"encoding/json"
	"reflect"
)

type RPCParamsCodec interface {
	Encode(values []reflect.Value) (json.RawMessage, error)
	Decode(rawValues json.RawMessage, valueTypes []reflect.Type) ([]reflect.Value, error)
}
