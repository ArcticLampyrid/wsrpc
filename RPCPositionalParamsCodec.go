package wsrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
)

type RPCPositionalParamsCodec struct {
}

func NewRPCPositionalParamsCodec() *RPCPositionalParamsCodec {
	return &RPCPositionalParamsCodec{}
}

func (*RPCPositionalParamsCodec) Encode(values []reflect.Value) (json.RawMessage, error) {
	nParams := len(values)
	result := make([]interface{}, nParams)
	for i := 0; i < nParams; i++ {
		result[i] = values[i].Interface()
	}
	return json.Marshal(result)
}

func (*RPCPositionalParamsCodec) Decode(rawValues json.RawMessage, valueTypes []reflect.Type) ([]reflect.Value, error) {
	if len(valueTypes) == 0 {
		// empty
		return []reflect.Value{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(rawValues))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	var valuesInJSON []json.RawMessage
	if t == nil {
		// JSON `null`
		valuesInJSON = []json.RawMessage{}
	} else if d, ok := t.(json.Delim); ok {
		if d.String() != "[" {
			return nil, errors.New("RPCPositionalParamsCodec can handle array and null only")
		}
		err = json.Unmarshal(rawValues, &valuesInJSON)
		if err != nil {
			return nil, err
		}
	}
	values := make([]reflect.Value, len(valueTypes))
	for i := 0; i < len(values); i++ {
		pType := valueTypes[i]
		if pType.Kind() == reflect.Ptr {
			values[i] = reflect.New(pType.Elem())
		} else {
			values[i] = reflect.New(pType)
		}
	}
	for i, curArg := range valuesInJSON {
		err = json.Unmarshal(curArg, values[i].Interface())
		if err != nil {
			return nil, err
		}
		pType := valueTypes[i]
		if pType.Kind() != reflect.Ptr {
			values[i] = values[i].Elem()
		}
	}
	return values, nil
}
