package filter

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fromJoinGrid struct {
	Id    int `db:"f.id"`
	TagId int `db:"t.id" grid:"skip,filter"`
}

func (T fromJoinGrid) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
	return qb.From("(SELECT ff.* FROM file ff LEFT JOIN tag as t ON ff.id = t.file_id) as f").
		LeftJoin("tag as t ON f.id = t.file_id").
		GroupBy("f.id")
}

func Test_FromJoinGrid(t *testing.T) {
	prepareTestData(t)

	dto := GridDto{
		Filter: [][]Filter{},
		Sorter: nil,
		Paging: Paging{},
		Search: "",
		Items:  nil,
	}

	var res []fromJoinGrid
	dto, err := GetData(fromJoinGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 2, dto.Paging.Total)

	dto = GridDto{
		Filter: [][]Filter{
			{
				{
					Column:   "tagId",
					Operator: "EQ",
					Value:    []string{"1"},
				},
			},
		},
		Sorter: nil,
		Paging: Paging{},
		Search: "",
		Items:  nil,
	}

	res = make([]fromJoinGrid, 0)
	dto, err = GetData(fromJoinGrid{}, dto, MariaDB, &res)
	require.Nil(t, err)

	assert.Equal(t, 1, dto.Paging.Total)
}
