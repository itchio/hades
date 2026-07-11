package hades

import (
	"fmt"
	"reflect"
)

type ScopeMap struct {
	byType   map[reflect.Type]*Scope
	byDBName map[string]*Scope
}

func NewScopeMap() *ScopeMap {
	return &ScopeMap{
		byType:   make(map[reflect.Type]*Scope),
		byDBName: make(map[string]*Scope),
	}
}

func (sm *ScopeMap) Add(c *Context, m any) error {
	val := reflect.ValueOf(m)

	if val.Type().Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Type().Kind() == reflect.Interface {
		val = val.Elem()
	}

	reflectType := val.Type()

	// what should we do if it's not a struct?
	if reflectType.Kind() != reflect.Struct {
		return fmt.Errorf("hades expects all models to be structs, but got %v instead", reflectType)
	}

	s := c.NewScope(m)
	sm.byType[reflect.PtrTo(reflectType)] = s
	sm.byDBName[s.TableName()] = s
	return nil
}

func (sm *ScopeMap) ByDBName(dbname string) *Scope {
	return sm.byDBName[dbname]
}

// ByModel returns the scope for a model, accepting pointer and value models
// alike — the same normalization Add applies when registering. Only the
// type is consulted, so a typed nil pointer is a valid descriptor. Returns
// nil for untyped nil and unregistered models.
func (sm *ScopeMap) ByModel(m any) *Scope {
	typ := reflect.TypeOf(m)
	if typ == nil {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return sm.byType[reflect.PtrTo(typ)]
}

func (sm *ScopeMap) ByType(typ reflect.Type) *Scope {
	return sm.byType[typ]
}

func (sm *ScopeMap) Each(f func(*Scope) error) error {
	for _, scope := range sm.byDBName {
		err := f(scope)
		if err != nil {
			return err
		}
	}
	return nil
}
