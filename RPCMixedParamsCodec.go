package wsrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
)

type RPCMixedParamsCodec struct {
	namedCodec      RPCNamedParamsCodec
	positionalCodec RPCPositionalParamsCodec
}

func NewRPCMixedParamsCodec(names []string) *RPCMixedParamsCodec {
	return &RPCMixedParamsCodec{
		namedCodec:      constructRPCNamedParamsCodec(names),
		positionalCodec: RPCPositionalParamsCodec{},
	}
}

func (c *RPCMixedParamsCodec) Encode(values []reflect.Value) (json.RawMessage, error) {
	return c.namedCodec.Encode(values)
}

func (c *RPCMixedParamsCodec) Decode(rawValues json.RawMessage, valueTypes []reflect.Type) ([]reflect.Value, error) {
	if len(valueTypes) == 0 {
		// empty
		return []reflect.Value{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(rawValues))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if t == nil {
		// JSON `null`
		return c.positionalCodec.Decode([]byte("[]"), valueTypes)
	}
	if d, ok := t.(json.Delim); ok {
		switch d.String() {
		case "{":
			return c.namedCodec.Decode(rawValues, valueTypes)
		case "[":
			return c.positionalCodec.Decode(rawValues, valueTypes)
		}
	}
	return nil, errors.New("RPCNamedParamsCodec can handle array and object and null only")
}

func (c *RPCMixedParamsCodec) AllowExcessive() bool {
	value1 := c.positionalCodec.AllowExcessive()
	value2 := c.namedCodec.AllowExcessive()
	if value1 != value2 {
		panic(errors.New("allow excessive mismatch for internal codecs"))
	}
	return value1
}

func (c *RPCMixedParamsCodec) WithAllowExcessive(allowExcessive bool) *RPCMixedParamsCodec {
	c.positionalCodec.WithAllowExcessive(allowExcessive)
	c.namedCodec.WithAllowExcessive(allowExcessive)
	return c
}
