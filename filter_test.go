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

const filterJSON = `[
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
]`

const jsonStr = `{
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
  "filter": ` + filterJSON + `
}`

const queryStr = `page=1&itemsPerPage=20&search=value&sort=firstName,ASC&sort=created,DESC&sort=id&filter=` + filterJSON

var expectedFilter = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: "value",
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

// This is custom type
type UUID struct {
	uuid.UUID
}

func (u UUID) Value() (driver.Value, error) {
	return u.MarshalBinary()
}

// This is example struct
type S struct {
	ID          UUID       `json:"id"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	Name        string     `json:"name"`
	Value       int        `json:"value"`
	ActivatedAt *time.Time `json:"activatedAt" db:"activated_at"`
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
	u, _ := url.Parse(`http://example.org`)
	u.RawQuery = queryStr

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

	if !reflect.DeepEqual(expectedFilter, filter) {
		t.Fatalf("expected: %+v, got: %+v", expectedFilter, filter)
	}

	// Fit filter to concrete type
	// remove unused filters, cast json names to db names and transform all registered types
	if err := filter.FitToModel(S{}); err != nil {
		t.Fatal(err)
	}
	// get SQL query
	var table, columns = "accounts", []string{"id", "created_at", "name", "value", "activated_at"}
	sql, args, err := filter.ToSql(table, columns...)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(sql)
	fmt.Println(args)

}

func Test_ParsePost(t *testing.T) {
	req := &http.Request{
		Method: http.MethodPost,
		Body:   ioutil.NopCloser(bytes.NewBufferString(jsonStr)),
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedFilter, filter) {
		t.Fatalf("expected: %+v, got: %+v", expectedFilter, filter)
	}

	// Fit filter to concrete type
	// remove unused filters, cast json names to db names and transform all registered types
	if err := filter.FitToModel(S{}); err != nil {
		t.Fatal(err)
	}
	// get SQL query
	var table, columns = "accounts", []string{"id", "created_at", "name", "value", "activated_at"}
	sql, args, err := filter.ToSql(table, columns...)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(sql)
	fmt.Println(args)

	//db, err := ConnForTest(test)
	//if err != nil {
	//	test.Fatal(err)
	//}
	//_ = sqlx.Select(db, &S{}, sql, args)
}
