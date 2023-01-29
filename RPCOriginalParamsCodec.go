package wsrpc

import (
	"encoding/json"
	"errors"
	"reflect"
)

type RPCOriginalParamsCodec struct {
}

func NewRPCOriginalParamsCodec() *RPCOriginalParamsCodec {
	return &RPCOriginalParamsCodec{}
}

func (*RPCOriginalParamsCodec) Encode(values []reflect.Value) (json.RawMessage, error) {
	nParams := len(values)
	if nParams == 0 {
		return jsonNullValue, nil
	} else if nParams != 0 {
		return nil, errors.New("original codec should be applied to 0 or 1 param only")
	}
	return json.Marshal(values[0].Interface())
}

func (*RPCOriginalParamsCodec) Decode(rawValues json.RawMessage, valueTypes []reflect.Type) ([]reflect.Value, error) {
	if len(valueTypes) == 0 {
		return []reflect.Value{}, nil
	} else if len(rawValues) != 0 {
		return nil, errors.New("original codec should be applied to 0 or 1 param only")
	}
	var value reflect.Value
	pType := valueTypes[0]
	if pType.Kind() == reflect.Ptr {
		value = reflect.New(pType.Elem())
	} else {
		value = reflect.New(pType)
	}
	err := json.Unmarshal(rawValues, value.Interface())
	if err != nil {
		return nil, err
	}
	return []reflect.Value{value}, nil
}
