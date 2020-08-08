package wsrpc

import (
	"encoding/json"
	"errors"
	"reflect"
)

type valueToJSONConverter struct {
	valueType  []reflect.Type
	name       []string
	nValue     int
	hasErrInfo bool
	flatten    bool
}

func newValueToJSONConverter(valueType []reflect.Type, name []string, flattenForSignle bool) *valueToJSONConverter {
	nValue := len(valueType)
	hasErrInfo := false
	if nValue > 0 && valueType[nValue-1] == typeOfError {
		hasErrInfo = true
		nValue--
	}
	if name != nil {
		if nValue != len(name) {
			panic(errors.New("length of param names must match length of params"))
		}
	}
	return &valueToJSONConverter{
		valueType:  valueType,
		name:       name,
		nValue:     nValue,
		hasErrInfo: hasErrInfo,
		flatten:    nValue == 1 && flattenForSignle,
	}
}

func (c *valueToJSONConverter) tryRun(values []reflect.Value) (json.RawMessage, error) {
	if c.hasErrInfo {
		errorOut := values[c.nValue].Interface()
		if errorOut != nil {
			return nil, errorOut.(error)
		}
	}
	if c.nValue == 0 {
		return jsonNullValue, nil
	}
	if c.name == nil {
		replyInJSON := make([]interface{}, c.nValue)
		for i := 0; i < c.nValue; i++ {
			replyInJSON[i] = values[i].Interface()
		}
		if c.flatten {
			return json.Marshal(replyInJSON[0])
		}
		return json.Marshal(replyInJSON)
	}
	namedReplyInJSON := make(map[string]interface{})
	for i := 0; i < c.nValue; i++ {
		namedReplyInJSON[c.name[i]] = values[i].Interface()
	}
	return json.Marshal(namedReplyInJSON)
}
