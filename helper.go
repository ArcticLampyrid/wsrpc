package wsrpc

import (
	"bytes"
	"encoding/json"
	"reflect"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var jsonNullValue = json.RawMessage([]byte("null"))

// IsJSONArray checks the input whether it is a JSON array or not.
func IsJSONArray(in []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(in))
	t, err := dec.Token()
	if err != nil {
		return false
	}
	if d, ok := t.(json.Delim); ok {
		switch d.String() {
		case "[":
			return true
		default:
			return false
		}
	}
	return false
}

// IsJSONObject checks the input whether it is a JSON object or not.
func IsJSONObject(in []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(in))
	t, err := dec.Token()
	if err != nil {
		return false
	}
	if d, ok := t.(json.Delim); ok {
		switch d.String() {
		case "{":
			return true
		default:
			return false
		}
	}
	return false
}

// IsJSONNull checks the input whether it is a JSON null value or not.
func IsJSONNull(in []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(in))
	t, err := dec.Token()
	if err != nil {
		return false
	}
	if t == nil {
		return true
	}
	return false
}

func getAllInParamInfo(fType reflect.Type) []reflect.Type {
	length := fType.NumIn()
	r := make([]reflect.Type, length)
	for i := 0; i < length; i++ {
		r[i] = fType.In(i)
	}
	return r
}

func getAllOutParamInfo(fType reflect.Type) []reflect.Type {
	length := fType.NumOut()
	r := make([]reflect.Type, length)
	for i := 0; i < length; i++ {
		r[i] = fType.Out(i)
	}
	return r
}
