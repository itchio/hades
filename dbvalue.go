package hades

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func DBValue(x any) (any, error) {
	typ := reflect.TypeOf(x)
	value := reflect.ValueOf(x)
	wasPtr := false

	if typ.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}

		wasPtr = true
		typ = typ.Elem()
		value = value.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return value.Interface().(time.Time).Format(time.RFC3339Nano), nil
		}
	case reflect.Slice:
		// stored as a JSON TEXT column, nil slices are NULL
		if value.IsNil() {
			return nil, nil
		}
		contents, err := json.Marshal(value.Interface())
		if err != nil {
			return nil, fmt.Errorf("encoding %v as JSON: %w", typ, err)
		}
		return string(contents), nil
	}

	if wasPtr {
		return value.Interface(), nil
	}
	return x, nil
}
