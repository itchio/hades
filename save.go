package hades

import (
	"fmt"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

type AllEntities map[reflect.Type]EntityMap
type EntityMap []any

func (c *Context) Save(conn *sqlite.Conn, rec any, opts ...SaveParam) (err error) {
	defer sqlitex.Save(conn)(&err)

	var params saveParams
	for _, o := range opts {
		o.ApplyToSaveParams(&params)
	}

	val := reflect.ValueOf(rec)
	valtyp := val.Type()
	if valtyp.Kind() == reflect.Slice {
		valtyp = valtyp.Elem()
	}
	if valtyp.Kind() != reflect.Pointer {
		return fmt.Errorf("Save expects a []*Model or a *Model, but it was passed a %v instead", val.Type())
	}

	riMap := make(RecordInfoMap)
	rootField := &assocField{
		name:     "<root>",
		mode:     AssocModeAppend,
		children: params.assocs,
	}
	rootRecordInfo, err := c.WalkType(riMap, rootField, valtyp)
	if err != nil {
		return fmt.Errorf("walking records to be saved: %w", err)
	}

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
		entities[typ] = append(entities[typ], v.Interface())
		return nil
	}

	var walk func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error

	var numVisited int64
	visit := func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error {
		if persist {
			if vri.Relationship != nil {
				switch vri.Relationship.Kind {
				case "has_many", "has_one":
					if len(pri.ModelStruct.PrimaryFields) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one primary key in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							p.Type(),
							len(pri.ModelStruct.PrimaryFields),
						)
					}
					pkField := p.Elem().FieldByName(pri.ModelStruct.PrimaryFields[0].Name)
					if len(vri.Relationship.ForeignFieldNames) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one foreign field in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.Relationship.ForeignFieldNames),
						)
					}
					fkField := v.Elem().FieldByName(vri.Relationship.ForeignFieldNames[0])
					fkField.Set(pkField)
				case "belongs_to":
					if len(vri.ModelStruct.PrimaryFields) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one primary key in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.ModelStruct.PrimaryFields),
						)
					}
					pkField := v.Elem().FieldByName(vri.ModelStruct.PrimaryFields[0].Name)

					if len(vri.Relationship.ForeignFieldNames) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one foreign field in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.Relationship.ForeignFieldNames),
						)
					}
					fkField := p.Elem().FieldByName(vri.Relationship.ForeignFieldNames[0])
					fkField.Set(pkField)
				case "many_to_many":
					vri.ManyToMany.Add(p, v)
				}
			}

			numVisited++
			err := addEntity(v)
			if err != nil {
				return fmt.Errorf("adding entity: %w", err)
			}
		}

		if v.Kind() != reflect.Pointer {
			return fmt.Errorf("expected a pointer, but got with %v", v)
		}
		vs := v.Elem()

		if vs.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct, but got with %v", v)
		}

		for _, childRi := range vri.Children {
			child := vs.FieldByName(childRi.Name())
			if !child.IsValid() {
				continue
			}

			if child.Kind() == reflect.Pointer && child.IsNil() {
				continue
			}

			// children are always saved
			persistChildren := true
			err := walk(v, vri, child, childRi, persistChildren)
			if err != nil {
				return fmt.Errorf("walking child entities to be saved: %w", err)
			}
		}
		return nil
	}

	walk = func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error {
		if v.Kind() == reflect.Slice {
			cull := false
			if vri.Relationship != nil {
				switch vri.Relationship.Kind {
				case "has_many":
					// if we're in replace mode
					if vri.Field.Mode() == AssocModeReplace {
						// and it's an actual has_many,
						// not a disguised many_to_many
						if len(vri.ModelStruct.PrimaryFields) == 1 {
							// then cull now
							cull = true
						}
					}
				case "many_to_many":
					// culling is done later, but let's record the ManyToMany now
					vri.ManyToMany.Mark(p)
				}
			}

			for i := 0; i < v.Len(); i++ {
				err := visit(p, pri, v.Index(i), vri, persist)
				if err != nil {
					return fmt.Errorf("walking slice of children: %w", err)
				}
			}

			if cull {
				rel := vri.Relationship

				parentPF := c.NewScope(p.Interface()).PrimaryField()
				if parentPF == nil {
					return fmt.Errorf("Can't save %v has_many %v: parent has no primary keys", pri.Type, vri.Type)
				}
				parentPK := parentPF.Field

				if len(vri.ModelStruct.PrimaryFields) != 1 {
					var pfNames []string
					for _, pf := range vri.ModelStruct.PrimaryFields {
						pfNames = append(pfNames, pf.Name)
					}

					return fmt.Errorf("Since %v has_many %v, expected %v to have one primary key. Instead, it has primary fields: %s",
						pri.Name(), vri.Name(), vri.Name(), strings.Join(pfNames, ", "))
				}

				valuePF := c.NewScope(v.Interface()).PrimaryField()
				if valuePF == nil {
					return fmt.Errorf("Can't save %v has_many %v: value has no primary keys", pri.Type, vri.Type)
				}

				// the cull identifies child rows by the CHILD's own primary
				// key (valuePF); rel.AssociationForeign* names the PARENT's
				// primary key and only coincidentally works when both are
				// named "id"
				passedPFs := make(map[any]struct{})
				for i := 0; i < v.Len(); i++ {
					rec := v.Index(i)
					pf := rec.Elem().FieldByName(valuePF.Name).Interface()
					passedPFs[pf] = struct{}{}
				}

				pfTyp := valuePF.Struct.Type
				pfKind := pfTyp.Kind()

				selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`,
					EscapeIdentifier(valuePF.DBName),
					EscapeIdentifier(vri.ModelStruct.TableName),
					EscapeIdentifier(rel.ForeignDBNames[0]),
				)
				deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND %s = ?`,
					EscapeIdentifier(vri.ModelStruct.TableName),
					EscapeIdentifier(rel.ForeignDBNames[0]),
					EscapeIdentifier(valuePF.DBName),
				)

				var removedPFs []any

				err := c.ExecRaw(conn, selectQuery, func(stmt *sqlite.Stmt) error {
					var pf any
					switch pfKind {
					case reflect.Int64:
						pf = stmt.ColumnInt64(0)
					case reflect.String:
						pf = stmt.ColumnText(0)
					default:
						return fmt.Errorf("Unsupported primary key for has_many: %v", pfTyp)
					}
					if _, ok := passedPFs[pf]; !ok {
						removedPFs = append(removedPFs, pf)
					}
					return nil
				}, parentPK.Interface())
				if err != nil {
					return err
				}

				for _, pf := range removedPFs {
					err := c.ExecRaw(conn, deleteQuery, nil, parentPK.Interface(), pf)
					if err != nil {
						return err
					}
				}
			}
		} else {
			err := visit(p, pri, v, vri, persist)
			if err != nil {
				return fmt.Errorf("walking single child: %w", err)
			}
		}
		return nil
	}

	err = walk(reflect.Zero(reflect.TypeOf(0)), nil, val, rootRecordInfo, !params.omitRoot)
	if err != nil {
		return fmt.Errorf("walking all records to be persisted: %w", err)
	}

	for typ, m := range entities {
		ri := riMap[typ]
		err := c.saveRows(conn, ri.Field.Mode(), m)
		if err != nil {
			return fmt.Errorf("saving rows: %w", err)
		}
	}

	for _, ri := range riMap {
		if ri.ManyToMany != nil {
			err := c.saveJoins(conn, ri.Field.Mode(), ri.ManyToMany)
			if err != nil {
				return fmt.Errorf("saving joins: %w", err)
			}
		}
	}

	return nil
}
