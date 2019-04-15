package filter

import (
	"bytes"
	"database/sql/driver"
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
			{Column: "id", Operator: "EQ", Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"}},
			{Column: "id", Operator: "NEQ", Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"}},
		}, {
			{Column: "value", Operator: "GT", Values: []interface{}{10.0}},
			{Column: "value", Operator: "LTE", Values: []interface{}{10.0}},
		}, {
			{Column: "value", Operator: "LT", Values: []interface{}{"10"}},
			{Column: "value", Operator: "GTE", Values: []interface{}{"10"}},
		}, {
			{Column: "name", Operator: "LIKE", Values: []interface{}{"John Smith"}},
		}, {
			{Column: "name", Operator: "STARTS", Values: []interface{}{"John"}},
			{Column: "name", Operator: "ENDS", Values: []interface{}{"Smith"}},
		}, {
			{Column: "activatedAt", Values: []interface{}{"2002-10-02T15:00:00Z"}},
			{Column: "activatedAt", Operator: "EMPTY"},
			{Column: "activatedAt", Operator: "NEMPTY"},
		}, {
			{Column: "createdAt", Operator: "BETWEEN", Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
			{Column: "createdAt", Operator: "NBETWEEN", Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
		},
	},
}
var filterFitted = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: search{value: "value", filters: [][]filter{
		{{column: "x.name", Operator: "LIKE", values: []interface{}{"value"}}},
	}},
	sorter: sorter{
		{Column: "firstName", Direction: "ASC"},
		{Column: "created", Direction: "DESC"},
		{Column: "id", column: "x.id"},
	},
	filter: [][]filter{
		{
			{Column: "id", column: "x.id", Operator: "EQ",
				Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"},
				values: []interface{}{UUID{UUID: uuid.MustParse("f1611454-debb-4d9f-bd78-83f0d38b0176")}},
			},
			{Column: "id", column: "x.id", Operator: "NEQ",
				Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"},
				values: []interface{}{UUID{UUID: uuid.MustParse("853649c7-9ff9-4572-b5b2-98f8da30e20a")}, UUID{UUID: uuid.MustParse("4b27dc87-e969-4bc3-afc5-195403fea580")}},
			},
		}, {
			{Column: "value", column: "x.value", Operator: "GT", Values: []interface{}{10.0}, values: []interface{}{10.0}},
			{Column: "value", column: "x.value", Operator: "LTE", Values: []interface{}{10.0}, values: []interface{}{10.0}},
		}, {
			{Column: "value", column: "x.value", Operator: "LT", Values: []interface{}{"10"}, values: []interface{}{"10"}},
			{Column: "value", column: "x.value", Operator: "GTE", Values: []interface{}{"10"}, values: []interface{}{"10"}},
		}, {
			{Column: "name", column: "x.name", Operator: "LIKE", Values: []interface{}{"John Smith"}, values: []interface{}{"John Smith"}},
		}, {
			{Column: "name", column: "x.name", Operator: "STARTS", Values: []interface{}{"John"}, values: []interface{}{"John"}},
			{Column: "name", column: "x.name", Operator: "ENDS", Values: []interface{}{"Smith"}, values: []interface{}{"Smith"}},
		}, {
			{Column: "activatedAt", column: "s.activated_at",
				Values: []interface{}{"2002-10-02T15:00:00Z"},
				values: []interface{}{time.Date(2002, 10, 2, 15, 00, 00, 0, time.UTC)}},
			{Column: "activatedAt", column: "s.activated_at", Operator: "EMPTY"},
			{Column: "activatedAt", column: "s.activated_at", Operator: "NEMPTY"},
		}, {
			{Column: "createdAt", column: "s.created_at", Operator: "BETWEEN",
				Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"},
				values: []interface{}{
					time.Date(2000, 10, 2, 15, 00, 00, 0, time.UTC),
					time.Date(2020, 10, 2, 15, 00, 00, 0, time.UTC),
				}},
			{Column: "createdAt", column: "s.created_at", Operator: "NBETWEEN",
				Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"},
				values: []interface{}{
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

	filter.FitToModel(model)

	if !reflect.DeepEqual(filterFitted, filter) {
		t.Fatalf("expected: \n%+v, got: \n%+v", filterFitted, filter)
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
