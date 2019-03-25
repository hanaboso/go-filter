package filter

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

const jsonStr = `{
  "paging": {
    "page": 1,
    "itemsPerPage": 20
  },
  "sorter": [
    {
      "column": "firstname",
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
        "column": "value",
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

func Test(test *testing.T) {
	// unmarshal test json
	var filter Body
	if err := json.Unmarshal([]byte(jsonStr), &filter); err != nil {
		test.Fatal(err)
	}
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
	// Fit filter to concrete type
	// remove unused filters, cast json names to db names and transform all registered types
	if err := filter.Filter.FitToModel(S{}); err != nil {
		test.Fatal(err)
	}
	// get SQL query
	var table, columns = "accounts", []string{"id", "created_at", "name", "value", "activated_at"}
	sql, args, err := filter.ToSql(table, columns...)
	if err != nil {
		test.Fatal(err)
	}

	fmt.Println(sql)
	fmt.Println(args)

	//db, err := ConnForTest(test)
	//if err != nil {
	//	test.Fatal(err)
	//}
	//_ = sqlx.Select(db, &S{}, sql, args)
}
