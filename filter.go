package filter

import (
	"encoding"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/Masterminds/squirrel"
)

var (
	DefaultLimit  uint64 = 10
	DefaultSorter        = sorter{{Column: "created", Direction: "DESC"}, {Column: "id", Direction: "DESC"}}
)

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
	} `json:"sorter,omitempty"`
	Cursor *struct {
		LastID       interface{} `json:"lastId"`
		ItemsPerPage uint64      `json:"itemsPerPage"`
	} `json:"cursoring,omitempty"`
	Filter [][]struct {
		Operator string        `json:"operator"`
		Column   string        `json:"column"`
		Values   []interface{} `json:"value"`
	} `json:"filter"`
}

// Filter
type Filter struct {
	filter filters
	paging paging
	search search
	sorter sorter

	fitted bool
	//m      *sync.Mutex
}

func (f Filter) AddFilter(column, operator string, values ...interface{}) Filter {
	f.filter = f.filter.AddFilter(column, operator, values)
	return f
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
	if resp.Paging.NextPage = resp.Paging.Page + 1; resp.Paging.NextPage > resp.Paging.LastPage {
		resp.Paging.NextPage = resp.Paging.LastPage
	}
	if resp.Paging.PreviousPage = resp.Paging.Page - 1; resp.Paging.PreviousPage < 1 {
		resp.Paging.PreviousPage = 1
	}
	resp.Search = f.search.value
	for _, s := range f.sorter {
		resp.Sorter = append(resp.Sorter, struct {
			Column    string `json:"column"`
			Direction string `json:"direction"`
		}{Column: s.Column, Direction: s.Direction})
	}
	resp.Filter = make([][]struct {
		Operator string        `json:"operator"`
		Column   string        `json:"column"`
		Values   []interface{} `json:"value"`
	}, len(f.filter))
	for i, filter := range f.filter {
		resp.Filter[i] = make([]struct {
			Operator string        `json:"operator"`
			Column   string        `json:"column"`
			Values   []interface{} `json:"value"`
		}, len(filter))
		for j, filter := range filter {
			resp.Filter[i][j].Column = filter.Column
			resp.Filter[i][j].Operator = filter.Operator
			resp.Filter[i][j].Values = filter.Values
		}
	}

	return resp
}

// MarshalJSON implements Marshaler interface
// anonymous struct is used for marshaling unexported fields
func (f Filter) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Filter filters `json:"filter"`
		Paging paging  `json:"paging"`
		Search string  `json:"search"`
		Sorter sorter  `json:"sorter,omitempty"`
	}{
		Filter: f.filter,
		Paging: f.paging,
		Search: f.search.value,
		Sorter: f.sorter,
	})
}

// UnmarshalJSON implements Unmarshaler interface
// anonymous struct is used for binding unexported fields
func (f *Filter) UnmarshalJSON(b []byte) error {
	var body struct {
		Filter filters `json:"filter"`
		Paging paging  `json:"paging"`
		Search string  `json:"search"`
		Sorter sorter  `json:"sorter"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return err
	}
	*f = Filter{
		filter: body.Filter,
		paging: body.Paging,
		search: search{value: body.Search},
		sorter: body.Sorter,
	}

	return nil
}

// FitToModel tries to fit filter to model, if error occurred filter will be in inappropriate state
func (f *Filter) FitToModel(model interface{}) {
	//f.m.Lock()
	//defer f.m.Unlock()
	if f.fitted {
		f.Reset()
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); f.filter.FitToModel(model) }()
	go func() { defer wg.Done(); f.search.FitToModel(model) }()
	go func() { defer wg.Done(); f.sorter.FitToModel(model) }()
	wg.Wait()
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

func (f *Filter) Reset() {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); f.filter.Reset() }()
	go func() { defer wg.Done(); f.search.Reset() }()
	go func() { defer wg.Done(); f.sorter.Reset() }()
	wg.Wait()
	f.fitted = false
}

// filters is type for all levels of filters
type filters [][]filter

func (f filters) AddFilter(column, operator string, values ...interface{}) filters {
	return append(f, []filter{{Column: column, Operator: operator, Values: values}})
}

// ToSql builds are nested filters to one Sql query
func (f filters) ToSql() (string, []interface{}, error) {
	var and squirrel.And
	for i := range f {
		var or squirrel.Or
		for j := range f[i] {
			or = append(or, f[i][j])
		}
		and = append(and, or)
	}
	return and.ToSql()
}

// FitToModel transforms all filters from JSON formats to DB format
// if some filter doesn't match with struct's field, than isn't marked as valid
func (f filters) FitToModel(model interface{}) {
	t := reflect.TypeOf(model)

	// change model type to non-pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// go over all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// fieldType is non-pointer type ex. time.Time
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		// fieldTypePtr is pointer type ex. *time.Time
		fieldTypePtr := reflect.New(fieldType)

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

		var wg sync.WaitGroup
		// change f type to same as in mapping struct
		for andI := range f {
			for orI := range f[andI] {
				wg.Add(1)
				go func(filter *filter) {
					defer wg.Done()
					// check if filter matches witch struct's json name
					if filter.Column != jsonName {
						return // skip this filter for this field
					}
					// create 1:1 copy of array
					filter.values = append(filter.Values[:0:0], filter.Values...)
					switch unmarshal := fieldTypePtr.Interface().(type) {
					case encoding.TextUnmarshaler:
						for i, v := range filter.Values {
							if err := unmarshal.UnmarshalText([]byte(v.(string))); err != nil {
								return // return to not save dbName to column, so filter wouldn't be used
							}
							filter.values[i] = reflect.Indirect(reflect.ValueOf(unmarshal)).Interface()
						}
					case json.Unmarshaler:
						for i, v := range filter.Values {
							s := `"` + v.(string) + `"`
							if err := unmarshal.UnmarshalJSON([]byte(s)); err != nil {
								return // return to not save dbName to column, so filter wouldn't be used
							}
							filter.values[i] = reflect.Indirect(reflect.ValueOf(unmarshal)).Interface()
						}
					default:
						for i, v := range filter.Values {
							filter.values[i] = v
						}
					}
					// column name is db name now
					filter.column = dbName

				}(&f[andI][orI])

			}
		}
		wg.Wait()
	}
}

func (f filters) Reset() {
	for i := range f {
		for j := range f[i] {
			f[i][j].Reset()
		}
	}
}

// filter is one concrete filter on one column
type filter struct {
	Operator string `json:"operator"`
	// json column name
	Column string `json:"column"`
	// database column name
	column string
	// json values
	Values []interface{} `json:"value"`
	// database values
	values []interface{}
}

// ToSql creates Sql from filter, but only if filter is marked as valid during filters.FitToModel
func (f filter) ToSql() (string, []interface{}, error) {
	if len(f.column) < 1 {
		return "", nil, nil
	}

	// TODO check length of f.Values without repetitions

	var sq squirrel.Sqlizer
	switch op := f.Operator; op {
	case "EQ", "":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Eq{f.column: f.values[0]}
	case "NEQ":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.NotEq{f.column: f.values[0]}
	case "GT":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Gt{f.column: f.values[0]}
	case "LT":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Lt{f.column: f.values[0]}
	case "GTE":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.GtOrEq{f.column: f.values[0]}
	case "LTE":
		if len(f.Values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.LtOrEq{f.column: f.values[0]}
	case "LIKE":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.column: fmt.Sprintf("%%%s%%", f.values[0])}
	case "STARTS":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.column: fmt.Sprintf("%%%s", f.values[0])}
	case "ENDS":
		if len(f.values) < 1 {
			return "", nil, fmt.Errorf("expected at least %d value for %s operator", 1, op)
		}
		sq = squirrel.Like{f.column: fmt.Sprintf("%s%%", f.values[0])}
	case "EMPTY":
		sq = squirrel.Eq{f.column: nil}
	case "NEMPTY":
		sq = squirrel.NotEq{f.column: nil}
	case "BETWEEN":
		if len(f.values) < 2 {
			return "", nil, fmt.Errorf("expected at least %d values for %s operator", 2, op)
		}
		sq = squirrel.And{
			squirrel.GtOrEq{f.column: f.values[0]},
			squirrel.Lt{f.column: f.values[1]},
		}
	case "NBETWEEN":
		if len(f.values) < 2 {
			return "", nil, fmt.Errorf("expected at least %d values for %s operator", 2, op)
		}
		sq = squirrel.Or{
			squirrel.Lt{f.column: f.values[0]},
			squirrel.GtOrEq{f.column: f.values[1]},
		}
	}

	return sq.ToSql()
}

func (f *filter) Reset() {
	f.column = ""
}

type search struct {
	value   string
	filters filters
}

func (s *search) FitToModel(model interface{}) {
	if len(s.value) < 1 {
		return
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

		if !isFieldSearchable(field) {
			continue
		}

		// get db name for this column
		dbName, ok := field.Tag.Lookup("db")
		if !ok {
			dbName = strings.ToLower(field.Name)
		}
		s.filters[0] = append(s.filters[0], filter{column: dbName, Operator: "LIKE", values: []interface{}{s.value}})
	}
}

func (s search) ToSql() (string, []interface{}, error) {
	return s.filters.ToSql()
}

func (s *search) Reset() {
	s.filters.Reset()
}

// sorter holds sorting rules
type sorter []struct {
	// json column name
	Column string `json:"column"`
	// database column name
	column    string
	Direction string `json:"direction"`
}

// OrderBy returns ORDER BY string for Sql
func (s sorter) OrderBy() []string {
	sl := make([]string, 0, len(s))
	for _, so := range s {
		if len(so.column) > 0 {
			sl = append(sl, fmt.Sprintf("%s %s", so.column, so.Direction))
		}
	}
	return sl
}

func (s sorter) Reset() {
	for i := range s {
		s[i].column = ""
	}
}

func (s *sorter) FitToModel(model interface{}) {
	if *s == nil {
		*s = DefaultSorter
	}

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

		// change f type to same as in mapping struct
		for i := range *s {
			if (*s)[i].Column == jsonName {
				// column name is db name now
				(*s)[i].column = dbName
			}
		}

	}
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
