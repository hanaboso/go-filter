package filter

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
)

// ConvertFunc is just typed function that's used for registered types for transforming data
type ConvertFunc func(i interface{}) (interface{}, error)

var registeredTypes = make(map[reflect.Type]ConvertFunc)

func init() {
	// register time.Time on init, RFC3339 layout is expected
	// this can't be replaced by re-registering time.Time type
	RegisterType(time.Time{}, func(layout string) ConvertFunc {
		return func(i interface{}) (i2 interface{}, e error) {
			s, ok := i.(string)
			if !ok {
				return nil, errors.New("expected string for time.Time field")
			}

			t, err := time.Parse(layout, s)
			if err != nil {
				return nil, fmt.Errorf("wrong time layout: %s, expected: %s", s, layout)
			}
			return t, nil
		}
	}(time.RFC3339))
}

// RegisterType is used for registering types in filter
// This registered types are transformed from API format to Db format by passed func
func RegisterType(i interface{}, f ConvertFunc) {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	registeredTypes[t] = f
}

// Body of filter
type Body struct {
	Filter Filters `json:"filter"`
	Paging Paging  `json:"paging"`
	Sorter Sorter  `json:"sorter"`
	LastID string  `json:"lastId"`
}

// ToSql returns sql, args and error build from contained filters
func (filter Body) ToSql(table string, columns ...string) (string, []interface{}, error) {
	return squirrel.
		Select(columns...).
		From(table).
		Where(filter.Filter).
		OrderBy(filter.Sorter.OrderBy()...).
		Limit(filter.Paging.Limit()).
		Offset(filter.Paging.Offset()).
		ToSql()
}

// Filters is type for all levels of filters
type Filters [][]Filter

// ToSql builds are nested filters to one Sql query
func (f Filters) ToSql() (string, []interface{}, error) {
	var and squirrel.And
	for _, f := range f {
		var or squirrel.Or
		for _, f := range f {
			or = append(or, f)
		}
		and = append(and, or)
	}
	return and.ToSql()
}

// FitToModel transforms all filters from JSON formats to DB format
// if some filter doesn't match with struct's field, than isn't marked as valid
func (f Filters) FitToModel(model interface{}) (err error) {
	t := reflect.TypeOf(model)

	// change model type to non-pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// go over all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// change type to non-pointer
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// get db name for this column
		dbName, ok := field.Tag.Lookup("db")
		if !ok {
			dbName = strings.ToLower(field.Name)
		}
		// get json name for this column
		jsonName, ok := field.Tag.Lookup("json")
		if !ok {
			jsonName = field.Name
		}

		// TODO goroutines?
		// change f type to same as in mapping struct
		for andI := range f {
			for orI := range f[andI] {
				filter := &f[andI][orI]
				// if filter match witch struct's json name
				if filter.Column == jsonName {
					// column name is db name now
					filter.Column = dbName
					// set as valid
					filter.valid = true

					if f, ok := registeredTypes[fieldType]; ok {
						for i, v := range filter.Value {
							filter.Value[i], err = f(v)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}

	}

	return nil
}

// Filter is one concrete filter on one column
type Filter struct {
	Operator string        `json:"operator"`
	Column   string        `json:"column"`
	Value    []interface{} `json:"value"`
	valid    bool
}

// ToSql creates Sql from filter, but only if filter is marked as valid during Filters.FitToModel
func (f Filter) ToSql() (string, []interface{}, error) {
	if !f.valid {
		return "", nil, nil
	}

	// TODO check length of f.Value without repetitions

	var sq squirrel.Sqlizer
	switch op := f.Operator; op {
	case "EQ", "":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Eq{f.Column: f.Value[0]}
	case "NEQ":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.NotEq{f.Column: f.Value[0]}
	case "GT":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Gt{f.Column: f.Value[0]}
	case "LT":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Lt{f.Column: f.Value[0]}
	case "GTE":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.GtOrEq{f.Column: f.Value[0]}
	case "LTE":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.LtOrEq{f.Column: f.Value[0]}
	case "LIKE":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.Column: f.Value[0]}
	case "STARTS":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.Column: fmt.Sprintf("%%%s", f.Value[0])}
	case "ENDS":
		if len(f.Value) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.Column: fmt.Sprintf("%s%%", f.Value[0])}
	case "EMPTY":
		sq = squirrel.Eq{f.Column: nil}
	case "NEMPTY":
		sq = squirrel.NotEq{f.Column: nil}
	case "BETWEEN":
		if len(f.Value) < 2 {
			return "", nil, fmt.Errorf("expected at least %d values for %s operator", 2, op)
		}
		sq = squirrel.And{
			squirrel.GtOrEq{f.Column: f.Value[0]},
			squirrel.Lt{f.Column: f.Value[1]},
		}
	case "NBETWEEN":
		if len(f.Value) < 2 {
			return "", nil, fmt.Errorf("expected at least %d values for %s operator", 2, op)
		}
		sq = squirrel.Or{
			squirrel.Lt{f.Column: f.Value[0]},
			squirrel.GtOrEq{f.Column: f.Value[1]},
		}
	}

	return sq.ToSql()
}

// Sorter holds sorting rules
type Sorter []struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

// OrderBy returns ORDER BY string for Sql
func (s Sorter) OrderBy() (sl []string) {
	for _, so := range s {
		sl = append(sl, fmt.Sprintf("%s %s", so.Column, so.Direction))
	}
	return sl
}

// Paging holds info for pagination
type Paging struct {
	Page         uint `json:"page"`
	ItemsPerPage uint `json:"itemsPerPage"`
}

// Limit returns LIMIT value for Sql
func (p Paging) Limit() uint64 {
	return uint64(p.ItemsPerPage)
}

// Offset returns OFFSET value for Sql
func (p Paging) Offset() uint64 {
	return uint64(p.Page-1) * p.Limit()
}
