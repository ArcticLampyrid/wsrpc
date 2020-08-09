package wsrpc

import (
	"encoding/json"
	"errors"
	"reflect"
)

type jsonToValueConverter struct {
	valueType        []reflect.Type
	nameToIndex      map[string]int
	hasInternalValue bool
	nSystemValue     int
	nValue           int
}

func newJSONToValueConverter(valueType []reflect.Type, internalValue reflect.Type, name []string) *jsonToValueConverter {
	nSystemValue := 0
	nValue := len(valueType)
	hasInternalValue := false
	var nameToIndex map[string]int
	if nValue > 0 && valueType[0] == internalValue {
		hasInternalValue = true
		nSystemValue++
		nValue--
	}
	if name != nil {
		if nValue != len(name) {
			panic(errors.New("length of param names must match length of params"))
		}
		nameToIndex = make(map[string]int)
		for i := 0; i < nValue; i++ {
			nameToIndex[name[i]] = i
		}
	}
	return &jsonToValueConverter{
		valueType:        valueType,
		nameToIndex:      nameToIndex,
		hasInternalValue: hasInternalValue,
		nSystemValue:     nSystemValue,
		nValue:           nValue,
	}
}

func (c *jsonToValueConverter) tryRun(internalValue interface{}, rawValues json.RawMessage) ([]reflect.Value, error) {
	var err error
	var ValuesInJSON []json.RawMessage
	if rawValues == nil || IsJSONNull(rawValues) {
		ValuesInJSON = make([]json.RawMessage, 0)
	} else if IsJSONArray(rawValues) {
		err := json.Unmarshal(rawValues, &ValuesInJSON)
		if err != nil {
			return nil, err
		}
	} else if IsJSONObject(rawValues) {
		if c.nameToIndex == nil {
			return nil, errors.New("no name information provided, cannot parse by-name arguments")
		}
		var namedValuesInJSON map[string]json.RawMessage
		err = json.Unmarshal(rawValues, &namedValuesInJSON)
		if err != nil {
			return nil, err
		}
		ValuesInJSON = make([]json.RawMessage, c.nValue)
		for key, value := range namedValuesInJSON {
			if i, ok := c.nameToIndex[key]; ok {
				ValuesInJSON[i] = value
			}
		}
	} else {
		ValuesInJSON = make([]json.RawMessage, 1)
		ValuesInJSON[0] = rawValues
	}
	Values := make([]reflect.Value, c.nSystemValue+c.nValue)
	if c.hasInternalValue {
		Values[0] = reflect.ValueOf(internalValue)
	}
	for i := c.nSystemValue; i < len(Values); i++ {
		pType := c.valueType[i]
		if pType.Kind() == reflect.Ptr {
			Values[i] = reflect.New(pType.Elem())
		} else {
			Values[i] = reflect.New(pType)
		}
	}
	for i, curArg := range ValuesInJSON {
		err = json.Unmarshal(curArg, Values[c.nSystemValue+i].Interface())
		if err != nil {
			return nil, err
		}
	}
	for i := c.nSystemValue; i < len(Values); i++ {
		pType := c.valueType[i]
		if pType.Kind() != reflect.Ptr {
			Values[i] = Values[i].Elem()
		}
	}
	return Values, nil
}
