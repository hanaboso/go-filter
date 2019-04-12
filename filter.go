package filter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"golang.org/x/sync/errgroup"
)

var (
	DefaultLimit  uint64 = 10
	DefaultSorter        = sorter{{Column: "created", Direction: "DESC"}, {Column: "id", Direction: "DESC"}}
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

type Resp struct {
	Items  interface{} `json:"items"`
	Paging struct {
		Page         uint64 `json:"page"`
		Total        uint64 `json:"total"`
		ItemsPerPage uint64 `json:"itemsPerPage"`
		LastPage     uint64 `json:"lastPage"`
		NextPage     uint64 `json:"nextPage"`
		PreviousPage uint64 `json:"previousPage"`
	} `json:"paging"`
	Search string `json:"search"`
	Sorter []struct {
		Column    string `json:"column"`
		Direction string `json:"direction"`
	} `json:"sorter"`
	LastID string `json:"lastId"`
	Filter [][]struct {
		Operator string        `json:"operator"`
		Column   string        `json:"column"`
		Value    []interface{} `json:"value"`
	} `json:"filter"`
}

// TODO validate input
// Filter
type Filter struct {
	filter filters
	paging paging
	search search
	sorter sorter
	lastID string
}

// TODO make immutable
func (f *Filter) AddFilter(column, operator string, values ...interface{}) {
	f.filter.AddFilter(column, operator, values)
}

func (f Filter) Resp(items interface{}, count uint64) (resp Resp) {
	resp.Items = items
	if resp.Paging.Page = f.paging.Page; resp.Paging.Page < 1 {
		resp.Paging.Page = 1
	}
	resp.Paging.Total = count
	resp.Paging.ItemsPerPage = f.paging.Limit()
	if resp.Paging.LastPage = resp.Paging.Total / resp.Paging.ItemsPerPage; resp.Paging.Total%resp.Paging.ItemsPerPage > 0 {
		resp.Paging.LastPage += 1
	}
	if resp.Paging.NextPage = f.paging.Page + 1; resp.Paging.NextPage > resp.Paging.LastPage {
		resp.Paging.NextPage = resp.Paging.LastPage
	}
	if resp.Paging.PreviousPage = f.paging.Page; resp.Paging.PreviousPage < 1 {
		resp.Paging.PreviousPage = 1
	}
	resp.Search = f.search.value
	for _, s := range f.sorter {
		resp.Sorter = append(resp.Sorter, struct {
			Column    string `json:"column"`
			Direction string `json:"direction"`
		}{Column: s.Column, Direction: s.Direction})
	}
	resp.LastID = f.lastID
	resp.Filter = make([][]struct {
		Operator string        `json:"operator"`
		Column   string        `json:"column"`
		Value    []interface{} `json:"value"`
	}, len(f.filter))
	for i, filter := range f.filter {
		resp.Filter[i] = make([]struct {
			Operator string        `json:"operator"`
			Column   string        `json:"column"`
			Value    []interface{} `json:"value"`
		}, len(filter))
		for j, filter := range filter {
			resp.Filter[i][j].Column = filter.Column
			resp.Filter[i][j].Operator = filter.Operator
			resp.Filter[i][j].Value = filter.Value
		}
	}

	return resp
}
func (f Filter) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Filter filters `json:"filter"`
		Paging paging  `json:"paging"`
		Search string  `json:"search"`
		Sorter sorter  `json:"sorter"`
		LastID string  `json:"lastId"`
	}{
		Filter: f.filter,
		Paging: f.paging,
		Search: f.search.value,
		Sorter: f.sorter,
		LastID: f.lastID,
	})
}

func (f *Filter) UnmarshalJSON(b []byte) error {
	var body struct {
		Filter filters `json:"filter"`
		Paging paging  `json:"paging"`
		Search string  `json:"search"`
		Sorter sorter  `json:"sorter"`
		LastID string  `json:"lastId"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return err
	}
	*f = Filter{
		filter: body.Filter,
		paging: body.Paging,
		search: search{value: body.Search},
		sorter: body.Sorter,
		lastID: body.LastID,
	}

	return nil
}

// FitToModel tries to fit filter to model, if error occurred filter will be in inappropriate state
// TODO mark filter as fitted, so it can be fitted only once
func (f *Filter) FitToModel(model interface{}) (err error) {
	var g errgroup.Group
	g.Go(func() error { return f.filter.FitToModel(model) })
	g.Go(func() error { return f.search.FitToModel(model) })
	g.Go(func() error { return f.sorter.FitToModel(model) })
	return g.Wait()
}

// ToSql returns sql, args and error build from contained filters
func (f Filter) ToSql(table string, columns ...string) (string, []interface{}, error) {
	return f.ExtendSelect(squirrel.Select(columns...).From(table)).ToSql()
}

func (f Filter) ExtendSelect(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
	return builder.
		Where(f.filter).
		Where(f.search).
		OrderBy(f.sorter.OrderBy()...).
		Limit(f.paging.Limit()).
		Offset(f.paging.Offset())
}

// filters is type for all levels of filters
type filters [][]filter

// TODO make immutable
func (f *filters) AddFilter(column, operator string, values ...interface{}) {
	*f = append(*f, []filter{{Column: column, Operator: operator, Value: values}})
}

// ToSql builds are nested filters to one Sql query
func (f filters) ToSql() (string, []interface{}, error) {
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
func (f filters) FitToModel(model interface{}) (err error) {
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

		// filter only by allowed fields
		if !isFieldFilterable(field) {
			continue
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
					// TODO try to use JSON/TEXT unmarshaler
					if f, ok := registeredTypes[fieldType]; ok {
						for i, v := range filter.Value {
							if filter.Value[i], err = f(v); err != nil {
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

// filter is one concrete filter on one column
type filter struct {
	Operator string        `json:"operator"`
	Column   string        `json:"column"`
	Value    []interface{} `json:"value"`
	valid    bool
}

// ToSql creates Sql from filter, but only if filter is marked as valid during filters.FitToModel
func (f filter) ToSql() (string, []interface{}, error) {
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
		sq = squirrel.Like{f.Column: fmt.Sprintf("%%%s%%", f.Value[0])}
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

type search struct {
	value   string
	filters filters
}

func (s *search) FitToModel(model interface{}) error {
	if len(s.value) < 1 {
		return nil
	}
	s.filters = make(filters, 1)

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

		// TODO check if searchable field
		if !isFieldSearchable(field) {
			continue
		}

		// get db name for this column
		dbName, ok := field.Tag.Lookup("db")
		if !ok {
			dbName = strings.ToLower(field.Name)
		}
		s.filters[0] = append(s.filters[0], filter{Column: dbName, Operator: "LIKE", Value: []interface{}{s.value}, valid: true})
	}

	return nil
}

func (s search) ToSql() (string, []interface{}, error) {
	return s.filters.ToSql()
}

// sorter holds sorting rules
type sorter []struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
	valid     bool
}

// OrderBy returns ORDER BY string for Sql
func (s sorter) OrderBy() []string {
	sl := make([]string, 0, len(s))
	for _, so := range s {
		if so.valid {
			sl = append(sl, fmt.Sprintf("%s %s", so.Column, so.Direction))
		}
	}
	return sl
}

// TODO DefaultSorter
func (s sorter) FitToModel(model interface{}) error {
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

		if !isFieldSortable(field) {
			continue
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
		for i := range s {
			if s[i].Column == jsonName {
				// column name is db name now
				s[i].Column = dbName
				// set as valid
				s[i].valid = true
			}
		}
	}

	return nil
}

// paging holds info for pagination
type paging struct {
	Page         uint64 `json:"page"`
	ItemsPerPage uint64 `json:"itemsPerPage"`
}

// Limit returns LIMIT value for Sql
func (p paging) Limit() uint64 {
	limit := p.ItemsPerPage
	if limit == 0 {
		limit = DefaultLimit
	}
	return limit
}

// Offset returns OFFSET value for Sql
func (p paging) Offset() uint64 {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * p.Limit()
}

// Parse parses request by method and returns filter
func Parse(req *http.Request) (Filter, error) {
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		return Filter{}, fmt.Errorf("wrong content-type: %s", ct)
	}

	switch {
	case req.Method == http.MethodGet:
		return parseGet(req)
	case req.Method == http.MethodPost:
		return parsePost(req)
	default:
		return Filter{}, fmt.Errorf("unknown method %s", req.Method)
	}
}

func parseGet(req *http.Request) (body Filter, err error) {
	q := req.URL.Query().Get("filter")
	if len(q) < 1 {
		return Filter{}, nil
	}
	if err := json.Unmarshal([]byte(q), &body); err != nil {
		return Filter{}, err
	}

	return body, nil
}

func parsePost(req *http.Request) (body Filter, err error) {
	// defer req.Filter.Close() // should i close it here?
	return body, json.NewDecoder(req.Body).Decode(&body)
}

func isField(method string, field reflect.StructField) bool {
	gridTag, ok := field.Tag.Lookup("grid")
	if !ok {
		return false
	}
	for _, tag := range strings.Split(gridTag, ",") {
		if tag == method {
			return true
		}
	}
	return false
}

func isFieldFilterable(field reflect.StructField) bool {
	return isField("filter", field)
}

func isFieldSearchable(field reflect.StructField) bool {
	return isField("search", field)
}

func isFieldSortable(field reflect.StructField) bool {
	return isField("sort", field)
}
