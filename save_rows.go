package hades

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"crawshaw.io/sqlite"
)

func (c *Context) saveRows(conn *sqlite.Conn, mode AssocMode, inputIface any) error {
	// inputIFace is a `[]any`
	input := reflect.ValueOf(inputIface)
	if input.Kind() != reflect.Slice {
		return errors.New("diff needs a slice")
	}

	if input.Len() == 0 {
		return nil
	}

	// we're trying to make fresh a `[]*SomeModel` slice, so
	// that scope can get annotation information from SomeModel struct annotations
	first := input.Index(0).Elem()
	fresh := reflect.MakeSlice(reflect.SliceOf(first.Type()), input.Len(), input.Len())
	for i := 0; i < input.Len(); i++ {
		record := input.Index(i).Elem()
		fresh.Index(i).Set(record)
	}

	// use scope to find the primary keys
	scope := c.NewScope(first.Interface())
	modelName := scope.GetModelStruct().ModelType.Name()
	primaryFields := scope.PrimaryFields()

	// this will happen for associations
	if len(primaryFields) != 1 {
		if len(primaryFields) != 2 {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(primaryFields), modelName)
		}

		recordsGroupedByPrimaryField := make(map[*Field]map[any][]reflect.Value)

		for _, primaryField := range primaryFields {
			recordsByKey := make(map[any][]reflect.Value)

			for i := 0; i < fresh.Len(); i++ {
				rec := fresh.Index(i)
				key := rec.Elem().FieldByName(primaryField.Name).Interface()
				recordsByKey[key] = append(recordsByKey[key], rec)
			}
			recordsGroupedByPrimaryField[primaryField] = recordsByKey
		}

		var bestSourcePrimaryField *Field
		var bestNumGroups int64 = math.MaxInt64
		var valueMap map[any][]reflect.Value
		for primaryField, recs := range recordsGroupedByPrimaryField {
			numGroups := len(recs)
			if int64(numGroups) < bestNumGroups {
				bestSourcePrimaryField = primaryField
				bestNumGroups = int64(numGroups)
				valueMap = recs
			}
		}

		if bestSourcePrimaryField == nil {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(primaryFields), modelName)
		}

		var bestDestinPrimaryField *Field
		for primaryField := range recordsGroupedByPrimaryField {
			if primaryField != bestSourcePrimaryField {
				bestDestinPrimaryField = primaryField
				break
			}
		}
		if bestDestinPrimaryField == nil {
			return errors.New("Internal error: could not find bestDestinPrimaryField")
		}

		sourceJTFK := JoinTableForeignKey{
			DBName:            ToDBName(bestSourcePrimaryField.Name),
			AssociationDBName: "<AssociationDBName left blank intentionally>",
		}

		destinJTFK := JoinTableForeignKey{
			DBName:            ToDBName(bestDestinPrimaryField.Name),
			AssociationDBName: "<AssociationDBName left blank intentionally>",
		}

		mtm, err := c.NewManyToMany(
			scope.TableName(),
			[]JoinTableForeignKey{sourceJTFK},
			[]JoinTableForeignKey{destinJTFK},
		)
		if err != nil {
			return fmt.Errorf("creating ManyToMany relationship: %w", err)
		}

		for sourceKey, recs := range valueMap {
			for _, rec := range recs {
				destinKey := rec.Elem().FieldByName(bestDestinPrimaryField.Name).Interface()
				mtm.AddKeys(sourceKey, destinKey, rec)
			}
		}

		err = c.saveJoins(conn, mode, mtm)
		if err != nil {
			return fmt.Errorf("saving joins: %w", err)
		}

		return nil
	}

	for i := 0; i < fresh.Len(); i++ {
		rec := fresh.Index(i)
		err := c.Upsert(conn, scope, rec)
		if err != nil {
			return fmt.Errorf("upserting DB records: %w", err)
		}
	}

	return nil
}
