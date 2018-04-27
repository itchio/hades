package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

func (c *Context) saveJoins(params *SaveParams, conn *sqlite.Conn, mtm *ManyToMany) error {
	cull := true
	for _, dc := range params.DontCull {
		if mtm.JoinTable == ToDBName(c.NewScope(dc).TableName()) {
			cull = false
			break
		}
	}

	joinType := reflect.PtrTo(mtm.Scope.GetModelStruct().ModelType)

	getDestinKey := func(v reflect.Value) interface{} {
		return v.Elem().FieldByName(mtm.DestinName).Interface()
	}

	for sourceKey, joinRecs := range mtm.Values {
		cacheAddr := reflect.New(reflect.SliceOf(joinType))

		err := c.Select(conn, cacheAddr.Interface(), builder.Eq{mtm.SourceDBName: sourceKey}, nil)
		if err != nil {
			return errors.Wrap(err, "fetching cached records to compare later")
		}

		cache := cacheAddr.Elem()

		cacheByDestinKey := make(map[interface{}]reflect.Value)
		for i := 0; i < cache.Len(); i++ {
			rec := cache.Index(i)
			cacheByDestinKey[getDestinKey(rec)] = rec
		}

		freshByDestinKey := make(map[interface{}]reflect.Value)
		for _, joinRec := range joinRecs {
			freshByDestinKey[joinRec.DestinKey] = joinRec.Record
		}

		var deletes []interface{}
		updates := make(map[interface{}]ChangedFields)
		var inserts []JoinRec

		// compare with cache: will result in delete or update
		for i := 0; i < cache.Len(); i++ {
			crec := cache.Index(i)
			destinKey := getDestinKey(crec)
			if frec, ok := freshByDestinKey[destinKey]; ok {
				if frec.IsValid() {
					// compare to maybe update
					ifrec := frec.Elem().Interface()
					icrec := crec.Elem().Interface()

					cf, err := DiffRecord(ifrec, icrec, mtm.Scope)
					if err != nil {
						return errors.Wrap(err, "diffing database records")
					}

					if cf != nil {
						updates[destinKey] = cf
					}
				}
			} else {
				deletes = append(deletes, destinKey)
			}
		}

		for _, joinRec := range joinRecs {
			if _, ok := cacheByDestinKey[joinRec.DestinKey]; !ok {
				inserts = append(inserts, joinRec)
			}
		}

		if !cull {
			// Not deleting extra join records, as requested
		} else {
			if len(deletes) > 0 {
				// FIXME: this needs to be paginated to avoid hitting SQLite max variables
				err := c.Exec(conn, builder.Delete(
					builder.Eq{mtm.SourceDBName: sourceKey},
					builder.In(mtm.DestinDBName, deletes...),
				).From(mtm.Scope.TableName()), nil)
				if err != nil {
					return errors.Wrap(err, "deleting extraneous relations")
				}
			}
		}

		for _, joinRec := range inserts {
			rec := joinRec.Record

			if !rec.IsValid() {
				// if not passed an explicit record, make it ourselves
				// that typically means the join table doesn't have additional
				// columns and is a simple many2many
				rec = reflect.New(joinType.Elem())
				rec.Elem().FieldByName(mtm.SourceName).Set(reflect.ValueOf(sourceKey))
				rec.Elem().FieldByName(mtm.DestinName).Set(reflect.ValueOf(joinRec.DestinKey))
			}

			// FIXME: that's slow/bad because of ToEq
			err := c.Insert(conn, mtm.Scope, rec)
			if err != nil {
				return errors.Wrap(err, "creating new relation records")
			}
		}

		for destinKey, rec := range updates {
			// FIXME: that's slow/bad
			eq := make(builder.Eq)
			for k, v := range rec {
				eq[ToDBName(k)] = v
			}
			err := c.Exec(conn, builder.Update(eq).Into(mtm.Scope.TableName()).Where(builder.Eq{mtm.SourceDBName: sourceKey, mtm.DestinDBName: destinKey}), nil)
			if err != nil {
				return errors.Wrap(err, "updating related records")
			}
		}
	}

	return nil
}
