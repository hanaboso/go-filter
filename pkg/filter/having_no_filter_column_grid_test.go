package filter

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type havingNoFilterColumnTableGrid struct {
	Id           int `db:"f.id"`
	ArticleCount int `json:"articleCount" db:"articleCount" grid:"filter,skip"`
}

func (T havingNoFilterColumnTableGrid) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
	return qb.From("file as f").
		LeftJoin("losos.article as a ON f.id = a.file_id").
		GroupBy("f.id")
}

func (T havingNoFilterColumnTableGrid) QueryCallbacks() map[string]QueryCallback {
	return map[string]QueryCallback{
		"articleCount": func(qb squirrel.SelectBuilder, field, operator string, values []string) squirrel.SelectBuilder {
			return qb.Having(FormQuery("COUNT(f.id)", operator, values, false))
		},
	}
}

func Test_HavingNoFilterColumnTableGrid(t *testing.T) {
	prepareTestData(t)

	dto := GridDto{
		Filter: [][]Filter{},
		Sorter: nil,
		Paging: Paging{},
		Search: "",
		Items:  nil,
	}

	var res []havingNoFilterColumnTableGrid
	dto, err := GetData(havingNoFilterColumnTableGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 2, dto.Paging.Total)

	dto = GridDto{
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

	res = make([]havingNoFilterColumnTableGrid, 0)
	dto, err = GetData(havingNoFilterColumnTableGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 1, dto.Paging.Total)
}
