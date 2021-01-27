package filter

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TGrid struct {
	Id           int      `db:"f.id"`
	TagId        *int     `db:"t.id"`
	TagName      []string `json:"tagName" db:"t.Name" grid:"filter,skip"`
	TagCount     int      `json:"tagCount" db:"tagCount" grid:"filter"`
	ArticleId    *int     `db:"a.id"`
	ArticleName  []string `json:"articleName" db:"a.Name" grid:"filter,skip"`
	ArticleCount int      `json:"articleCount" db:"articleCount" grid:"filter"`
}

func (T TGrid) QueryCallbacks() map[string]QueryCallback {
	return map[string]QueryCallback{
		"articleCount": func(qb squirrel.SelectBuilder, field, operator string, values []string) squirrel.SelectBuilder {
			return qb.Having(FormQuery("articleCount", operator, values, false))
		},
	}
}

func (T TGrid) FilterCallbacks() map[string]FilterCallback {
	return map[string]FilterCallback{
		"tagCount": func(field, operator string, values []string) squirrel.Sqlizer {
			return FormQuery("COUNT(t.id) as tagCount", operator, values, false)
		},
	}
}

func (T TGrid) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
	return qb.
		Column("(SELECT COUNT(id) FROM losos.article WHERE file_id = f.id) articleCount").
		Column("(SELECT COUNT(id) FROM tag WHERE file_id = f.id) as tagCount").
		From("file as f").
		LeftJoin("tag as t ON f.id = t.file_id").
		LeftJoin("losos.article as a ON f.id = a.file_id").
		GroupBy("f.id")
}

func Test_FullGrid(t *testing.T) {
	prepareTestData(t)

	dto := GridDto{
		Filter: [][]Filter{
			{
				{
					Column:   "articleCount",
					Operator: "GTE",
					Value:    []string{"3"},
				},
			},
		},
		Sorter: nil,
		Paging: Paging{},
		Search: "",
		Items:  nil,
	}

	var res []TGrid
	dto, err := GetData(TGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 1, dto.Paging.Total)
	assert.Equal(t, 2, res[0].TagCount)
	assert.Equal(t, 3, res[0].ArticleCount)

	dto = GridDto{
		Filter: [][]Filter{
			{
				{
					Column:   "articleCount",
					Operator: "LTE",
					Value:    []string{"6"},
				},
			},
		},
		Sorter: nil,
		Paging: Paging{},
		Search: "",
		Items:  nil,
	}

	res = make([]TGrid, 0)
	dto, err = GetData(TGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 2, dto.Paging.Total)
}
