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

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/hanaboso/go-filter/pkg/filter/testdata"
	"github.com/stretchr/testify/assert"
)

const checkSQL = "SELECT x.id, s.created_at, x.name, x.value, s.activated_at FROM eska s JOIN ixka x ON x.name = s.name WHERE ((x.id = ? OR x.id <> ?) AND (x.value > ? OR x.value <= ?) AND (x.value < ? OR x.value >= ?) AND (x.name LIKE ?) AND (x.name LIKE ? OR x.name LIKE ?) AND (s.activated_at = ? OR s.activated_at IS NULL OR s.activated_at IS NOT NULL) AND ((s.created_at >= ? AND s.created_at < ?) OR (s.created_at < ? OR s.created_at >= ?)) AND (x.id IN (?) OR x.id NOT IN (?,?))) AND ((x.name LIKE ?)) ORDER BY x.id  LIMIT 20 OFFSET 0"

// This is custom type
type UUID struct {
	uuid.UUID
}

func (u UUID) Value() (driver.Value, error) {
	return u.MarshalBinary()
}

type TestModelFlat struct {
	ID          UUID       `json:"id" db:"x.id" grid:"filter,sort"`
	CreatedAt   time.Time  `json:"createdAt" db:"s.created_at" grid:"filter"`
	Name        string     `json:"name" db:"x.name" grid:"filter,search"`
	Value       int        `json:"value" db:"x.value" grid:"filter"`
	ActivatedAt *time.Time `json:"activatedAt,omitempty" db:"s.activated_at" grid:"filter"`
}

type TestModelNested struct {
	S struct {
		CreatedAt   time.Time  `json:"createdAt" db:"created_at" grid:"filter"`
		ActivatedAt *time.Time `json:"activatedAt,omitempty" db:"activated_at" grid:"filter"`
	} `json:"s" db:"s" grid:"dive"`
	X struct {
		ID    UUID   `json:"id" db:"id" grid:"filter,sort"`
		Name  string `json:"name" db:"name" grid:"filter,search"`
		Value int    `json:"value" db:"value" grid:"filter"`
	} `json:"x" db:"x" grid:"dive"`
}

func Test_ParseGet(t *testing.T) {
	b, err := testdata.Asset("filter_flat.json")
	if err != nil {
		t.Fatal(err)
	}

	u, _ := url.Parse(`http://httpbin.org/anything`)
	u.RawQuery = url.Values{"filter": {string(b)}}.Encode()

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
	b, err := testdata.Asset("filter_flat.json")
	if err != nil {
		t.Fatal(err)
	}
	rc := ioutil.NopCloser(bytes.NewReader(b))

	req := &http.Request{
		Method: http.MethodPost,
		Body:   rc,
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

func TestFilter_FitToModel_Flat(t *testing.T) {
	b, err := testdata.Asset("filter_flat.json")
	if err != nil {
		t.Fatal(err)
	}
	rc := ioutil.NopCloser(bytes.NewReader(b))

	req := &http.Request{
		Method: http.MethodPost,
		Body:   rc,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	var model TestModelFlat

	filter.FitToModel(model)

	if !reflect.DeepEqual(fittedFlatModel, filter) {
		t.Fatalf("expected: \n%+v, got: \n%+v", fittedFlatModel, filter)
	}

	sq := squirrel.
		Select("x.id", "s.created_at", "x.name", "x.value", "s.activated_at").
		From("eska s").
		Join("ixka x ON x.name = s.name")

	sql, args, err := filter.ExtendSelect(sq).ToSql()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, args, 18)
	assert.Equal(t, checkSQL, sql)
}

func TestFilter_FitToModel_Nested(t *testing.T) {
	b, err := testdata.Asset("filter_nested.json")
	if err != nil {
		t.Fatal(err)
	}
	rc := ioutil.NopCloser(bytes.NewReader(b))

	req := &http.Request{
		Method: http.MethodPost,
		Body:   rc,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	var model TestModelNested

	filter.FitToModel(model)

	if !reflect.DeepEqual(fittedNestedModel, filter) {
		t.Fatalf("expected: \n%+v, got: \n%+v", fittedNestedModel, filter)
	}

	sq := squirrel.
		Select("x.id", "s.created_at", "x.name", "x.value", "s.activated_at").
		From("eska s").
		Join("ixka x ON x.name = s.name")

	sql, args, err := filter.ExtendSelect(sq).ToSql()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, args, 18)
	assert.Equal(t, checkSQL, sql)
}

func TestFilter_FitToModel_Embedded(t *testing.T) {
	b, err := testdata.Asset("filter_flat.json")
	if err != nil {
		t.Fatal(err)
	}
	rc := ioutil.NopCloser(bytes.NewReader(b))

	req := &http.Request{
		Method: http.MethodPost,
		Body:   rc,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	filter, err := Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	var model struct {
		TestModelFlat `grid:"dive"`
	}

	filter.FitToModel(model)

	if !reflect.DeepEqual(fittedFlatModel, filter) {
		t.Fatalf("expected: \n%+v, got: \n%+v", fittedFlatModel, filter)
	}

	sq := squirrel.
		Select("x.id", "s.created_at", "x.name", "x.value", "s.activated_at").
		From("eska s").
		Join("ixka x ON x.name = s.name")

	sql, args, err := filter.ExtendSelect(sq).ToSql()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, args, 18)
	assert.Equal(t, checkSQL, sql)
}
