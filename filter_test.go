package filter

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

const filterJSON = `{
  "paging": {
    "page": 1,
    "itemsPerPage": 20
  },
  "sorter": [
    {
      "column": "firstName",
      "direction": "ASC"
    },
    {
      "column": "created",
      "direction": "DESC"
    },
    {
      "column": "id"
    }
  ],
  "search": "value",
  "filter": [
[
  {
	"column": "id",
	"operator": "EQ",
	"value": [
	  "f1611454-debb-4d9f-bd78-83f0d38b0176"
	]
  },
  {
	"column": "id",
	"operator": "NEQ",
	"value": [
	  "853649c7-9ff9-4572-b5b2-98f8da30e20a",
	  "4b27dc87-e969-4bc3-afc5-195403fea580"
	]
  }
],
[
  {
	"column": "value",
	"operator": "GT",
	"value": [
	  10
	]
  },
  {
	"column": "value",
	"operator": "LTE",
	"value": [
	  10
	]
  }
],
[
  {
	"column": "value",
	"operator": "LT",
	"value": [
	  "10"
	]
  },
  {
	"column": "value",
	"operator": "GTE",
	"value": [
	  "10"
	]
  }
],
[
  {
	"column": "name",
	"operator": "LIKE",
	"value": [
	  "John Smith"
	]
  }
],
[
  {
	"column": "name",
	"operator": "STARTS",
	"value": [
	  "John"
	]
  },
  {
	"column": "name",
	"operator": "ENDS",
	"value": [
	  "Smith"
	]
  }
],
[
  {
	"column": "activatedAt",
	"value": [
	  "2002-10-02T15:00:00Z"
	]
  },
  {
	"column": "activatedAt",
	"operator": "EMPTY"
  },
  {
	"column": "activatedAt",
	"operator": "NEMPTY"
  }
],
[
  {
	"column": "createdAt",
	"operator": "BETWEEN",
	"value": [
	  "2000-10-02T15:00:00Z",
	  "2020-10-02T15:00:00Z"
	]
  },
  {
	"column": "createdAt",
	"operator": "NBETWEEN",
	"value": [
	  "2000-10-02T15:00:00Z",
	  "2020-10-02T15:00:00Z"
	]
  }
]
]
}`

var filterParsed = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: search{value: "value"},
	sorter: sorter{
		{Column: "firstName", Direction: "ASC"},
		{Column: "created", Direction: "DESC"},
		{Column: "id"},
	},
	filter: [][]filter{
		{
			{Column: "id", Operator: "EQ", Value: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"}},
			{Column: "id", Operator: "NEQ", Value: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"}},
		}, {
			{Column: "value", Operator: "GT", Value: []interface{}{10.0}},
			{Column: "value", Operator: "LTE", Value: []interface{}{10.0}},
		}, {
			{Column: "value", Operator: "LT", Value: []interface{}{"10"}},
			{Column: "value", Operator: "GTE", Value: []interface{}{"10"}},
		}, {
			{Column: "name", Operator: "LIKE", Value: []interface{}{"John Smith"}},
		}, {
			{Column: "name", Operator: "STARTS", Value: []interface{}{"John"}},
			{Column: "name", Operator: "ENDS", Value: []interface{}{"Smith"}},
		}, {
			{Column: "activatedAt", Value: []interface{}{"2002-10-02T15:00:00Z"}},
			{Column: "activatedAt", Operator: "EMPTY"},
			{Column: "activatedAt", Operator: "NEMPTY"},
		}, {
			{Column: "createdAt", Operator: "BETWEEN", Value: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
			{Column: "createdAt", Operator: "NBETWEEN", Value: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
		},
	},
}
var filterFitted = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: search{value: "value", filters: [][]filter{
		{{Column: "x.name", Operator: "LIKE", Value: []interface{}{"value"}, valid: true}},
	}},
	sorter: sorter{
		{Column: "firstName", Direction: "ASC"},
		{Column: "created", Direction: "DESC"},
		{Column: "x.id", valid: true},
	},
	filter: [][]filter{
		{
			{Column: "x.id", Operator: "EQ", Value: []interface{}{UUID{UUID: uuid.MustParse("f1611454-debb-4d9f-bd78-83f0d38b0176")}}, valid: true},
			{Column: "x.id", Operator: "NEQ", Value: []interface{}{UUID{UUID: uuid.MustParse("853649c7-9ff9-4572-b5b2-98f8da30e20a")}, UUID{UUID: uuid.MustParse("4b27dc87-e969-4bc3-afc5-195403fea580")}}, valid: true},
		}, {
			{Column: "x.value", Operator: "GT", Value: []interface{}{10.0}, valid: true},
			{Column: "x.value", Operator: "LTE", Value: []interface{}{10.0}, valid: true},
		}, {
			{Column: "x.value", Operator: "LT", Value: []interface{}{"10"}, valid: true},
			{Column: "x.value", Operator: "GTE", Value: []interface{}{"10"}, valid: true},
		}, {
			{Column: "x.name", Operator: "LIKE", Value: []interface{}{"John Smith"}, valid: true},
		}, {
			{Column: "x.name", Operator: "STARTS", Value: []interface{}{"John"}, valid: true},
			{Column: "x.name", Operator: "ENDS", Value: []interface{}{"Smith"}, valid: true},
		}, {
			{Column: "s.activated_at", Value: []interface{}{time.Date(2002, 10, 2, 15, 00, 00, 0, time.UTC)}, valid: true},
			{Column: "s.activated_at", Operator: "EMPTY", valid: true},
			{Column: "s.activated_at", Operator: "NEMPTY", valid: true},
		}, {
			{Column: "s.created_at", Operator: "BETWEEN", valid: true, Value: []interface{}{
				time.Date(2000, 10, 2, 15, 00, 00, 0, time.UTC),
				time.Date(2020, 10, 2, 15, 00, 00, 0, time.UTC),
			}},
			{Column: "s.created_at", Operator: "NBETWEEN", valid: true, Value: []interface{}{
				time.Date(2000, 10, 2, 15, 00, 00, 0, time.UTC),
				time.Date(2020, 10, 2, 15, 00, 00, 0, time.UTC),
			}},
		},
	},
}

// This is custom type
type UUID struct {
	uuid.UUID
}

func (u UUID) Value() (driver.Value, error) {
	return u.MarshalBinary()
}

func TestMain(m *testing.M) {
	// Register custom type for transforming API format to DB format
	RegisterType(UUID{}, func(i interface{}) (i2 interface{}, e error) {
		s, ok := i.(string)
		if !ok {
			return nil, errors.New("expected string for UUID field")
		}

		uuid, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("wrong format[%s] for UUID", s)
		}
		return UUID{UUID: uuid}, err
	})
	m.Run()
}

func Test_ParseGet(t *testing.T) {
	u, _ := url.Parse(`http://httpbin.org/anything`)
	u.RawQuery = url.Values{"filter": {filterJSON}}.Encode()

	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(filterParsed, filter) {
		t.Fatalf("expected: %+v, got: %+v", filterParsed, filter)
	}
}

func Test_ParsePost(t *testing.T) {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   ioutil.NopCloser(bytes.NewBufferString(filterJSON)),
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(filterParsed, filter) {
		t.Fatalf("expected: %+v, got: %+v", filterParsed, filter)
	}
}

func TestFilter_FitToModel(t *testing.T) {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   ioutil.NopCloser(bytes.NewBufferString(filterJSON)),
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	var model struct {
		ID          UUID       `json:"id" db:"x.id" grid:"filter,sort"`
		CreatedAt   time.Time  `json:"createdAt" db:"s.created_at" grid:"filter"`
		Name        string     `json:"name" db:"x.name" grid:"filter,search"`
		Value       int        `json:"value" db:"x.value" grid:"filter"`
		ActivatedAt *time.Time `json:"activatedAt" db:"s.activated_at" grid:"filter"`
	}

	if err := filter.FitToModel(model); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(filterFitted, filter) {
		t.Fatalf("expected: %+v, got: %+v", filterFitted, filter)
	}

	//sq := squirrel.
	//	Select("x.id", "s.created_at", "x.name", "x.value", "s.activated_at").
	//	From("eska s").
	//	Join("ixka x ON x.name = s.name")
	//
	//sql, args, err := filter.ExtendSelect(sq).ToSql()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//fmt.Println(sql)
	//fmt.Println(args)
}
