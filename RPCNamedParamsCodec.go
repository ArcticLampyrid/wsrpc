package wsrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
)

type RPCNamedParamsCodec struct {
	names          []string
	nameToID       map[string]int
	allowExcessive bool
}

func constructRPCNamedParamsCodec(names []string) RPCNamedParamsCodec {
	nameToID := make(map[string]int)
	for i, name := range names {
		nameToID[name] = i
	}
	return RPCNamedParamsCodec{
		names,
		nameToID,
		true,
	}
}

func NewRPCNamedParamsCodec(names []string) *RPCNamedParamsCodec {
	nameToID := make(map[string]int)
	for i, name := range names {
		nameToID[name] = i
	}
	return &RPCNamedParamsCodec{
		names,
		nameToID,
		true,
	}
}

func (c *RPCNamedParamsCodec) Encode(values []reflect.Value) (json.RawMessage, error) {
	nParams := len(values)
	if nParams != len(c.names) {
		return nil, errors.New("length of param names must match length of params")
	}
	result := make(map[string]interface{})
	for i := 0; i < nParams; i++ {
		result[c.names[i]] = values[i].Interface()
	}
	return json.Marshal(result)
}

func (c *RPCNamedParamsCodec) Decode(rawValues json.RawMessage, valueTypes []reflect.Type) ([]reflect.Value, error) {
	if len(valueTypes) == 0 {
		// empty
		return []reflect.Value{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(rawValues))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	var namedValuesInJSON map[string]json.RawMessage
	if t == nil {
		// JSON `null`
		namedValuesInJSON = make(map[string]json.RawMessage)
	} else if d, ok := t.(json.Delim); ok {
		if d.String() != "{" {
			return nil, errors.New("RPCNamedParamsCodec can handle object and null only")
		}
		err = json.Unmarshal(rawValues, &namedValuesInJSON)
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
	for key, value := range namedValuesInJSON {
		i, ok := c.nameToID[key]
		if !ok {
			if !c.allowExcessive {
				return nil, errors.New("too many arguments")
			}
			continue
		}
		err = json.Unmarshal(value, values[i].Interface())
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(values); i++ {
		pType := valueTypes[i]
		if pType.Kind() != reflect.Ptr {
			values[i] = values[i].Elem()
		}
	}
	return values, nil
}

func (c *RPCNamedParamsCodec) AllowExcessive() bool {
	return c.allowExcessive
}

func (c *RPCNamedParamsCodec) WithAllowExcessive(allowExcessive bool) *RPCNamedParamsCodec {
	c.allowExcessive = allowExcessive
	return c
}
