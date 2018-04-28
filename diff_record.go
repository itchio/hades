package hades

import (
	"reflect"
	"time"

	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

type ChangedFields map[*StructField]interface{}

func DiffRecord(x, y interface{}, scope *Scope) (ChangedFields, error) {
	if x == nil || y == nil {
		return nil, errors.New("DiffRecord: arguments must not be nil")
	}
	// v1 is the fresh record (being saved)
	v1 := reflect.ValueOf(x)
	// v2 is the cached record (in DB)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return nil, errors.New("DiffRecord: arguments are not the same type")
	}

	typ := v1.Type()
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("DiffRecord: arguments must be structs")
	}

	ms := scope.GetModelStruct()
	var res ChangedFields
	for i, n := 0, v1.NumField(); i < n; i++ {
		f := typ.Field(i)
		fieldName := f.Name

		sf, ok := ms.StructFieldsByName[fieldName]
		if !ok {
			// not listed as a field? ignore
			continue
		}

		if !sf.IsNormal {
			continue
		}

		v1f := v1.Field(i)
		v2f := v2.Field(i)

		iseq, err := iseq(sf, v1f, v2f)
		if err != nil {
			return res, err
		}

		if !iseq {
			if res == nil {
				res = make(ChangedFields)
			}
			res[sf] = v1f.Interface()
		}
	}

	return res, nil
}

func iseq(sf *StructField, v1f reflect.Value, v2f reflect.Value) (bool, error) {
	typ := sf.Struct.Type
	originalTyp := typ

	if typ.Kind() == reflect.Ptr {
		if v1f.IsNil() {
			if !v2f.IsNil() {
				return false, nil // only v1 nil
			}
			return true, nil // both nil
		} else {
			if v2f.IsNil() {
				return false, nil // only v2 nil
			}

			// neither are nil, let's compare values
			typ = typ.Elem()
			v1f = v1f.Elem()
			v2f = v2f.Elem()
		}
	}

	switch typ.Kind() {
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
		reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		eq := v1f.Int() == v2f.Int()
		return eq, nil
	case reflect.Bool:
		eq := v1f.Bool() == v2f.Bool()
		return eq, nil
	case reflect.Float64, reflect.Float32:
		eq := v1f.Float() == v2f.Float()
		return eq, nil
	case reflect.String:
		eq := v1f.String() == v2f.String()
		return eq, nil
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			eq := v1f.Interface().(time.Time).UnixNano() == v2f.Interface().(time.Time).UnixNano()
			return eq, nil
		}
	}

	return false, errors.Errorf("Don't know how to compare fields of type %v", originalTyp)
}

func (cf ChangedFields) ToEq() builder.Eq {
	eq := make(builder.Eq)
	for sf, v := range cf {
		eq[sf.DBName] = DBValue(v)
	}
	return eq
}
