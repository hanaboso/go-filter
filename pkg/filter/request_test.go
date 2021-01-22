package filter

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateGridDto(t *testing.T) {
	queries := []string{
		"_page=5",
		"_size=50",
		"_search=search",
		"_filter:col2:NEQ:1=a",
		"_filter:col:EQ=asd,qwe",
		"_sorter:cc=ASC",
	}

	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("/?%s", strings.Join(queries, "&")),
		bytes.NewReader([]byte{}),
	)

	dto := CreateGridDto(req)
	exp := GridDto{
		Filter: [][]Filter{
			{
				{
					Column:   "col",
					Operator: "EQ",
					Value:    []string{"asd", "qwe"},
				},
			},
			{
				{
					Column:   "col2",
					Operator: "NEQ",
					Value:    []string{"a"},
				},
			},
		},
		Sorter: []Sorter{
			{
				Column:    "cc",
				Direction: "ASC",
			},
		},
		Paging: Paging{
			Page: 5,
			Size: 50,
		},
		Search: "search",
	}

	assert.Equal(t, exp, dto)
}
