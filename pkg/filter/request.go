package filter

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	search      = "_search"
	page        = "_page"
	size        = "_size"
	sorter      = "_sorter"
	filter      = "_filter"
	defaultSize = 10
)

type GridDto struct {
	Filter [][]Filter  `json:"filter"`
	Sorter []Sorter    `json:"sorter"`
	Paging Paging      `json:"paging"`
	Search string      `json:"search"`
	Items  interface{} `json:"items"`
}

type Filter struct {
	Column   string   `json:"column"`
	Operator string   `json:"operator"`
	Value    []string `json:"value"`
}

type Sorter struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type Paging struct {
	Page         int `json:"page"`
	Size         int `json:"size"`
	LastPage     int `json:"lastPage"`
	NextPage     int `json:"nextPage"`
	PreviousPage int `json:"previousPage"`
	Total        int `json:"total"`
}

func CreateGridDto(request *http.Request) GridDto {
	dto := GridDto{
		Filter: make([][]Filter, 0),
		Sorter: make([]Sorter, 0),
		Paging: Paging{
			Size: defaultSize,
			Page: 1,
		},
	}

	values := request.URL.Query()

	for key := range values {
		if key == search {
			dto.Search = values.Get(key)
			continue
		}

		if key == page {
			dto.Paging.Page = intVal(values.Get(page))
			if dto.Paging.Page <= 0 {
				dto.Paging.Page = 1
			}

			continue
		}

		if key == size {
			dto.Paging.Size = intVal(values.Get(size))
			if dto.Paging.Size <= 0 {
				dto.Paging.Size = defaultSize
			}

			continue
		}

		// _sorter:column:optIndex=direction
		if strings.HasPrefix(key, sorter) {
			parts := strings.Split(key, ":")
			if len(parts) < 2 {
				continue
			}
			column := parts[1]
			index := 0

			if len(parts) >= 3 {
				index = intVal(parts[2])
			}

			dto = extendSorter(dto, index)
			dto.Sorter[index] = Sorter{
				Column:    column,
				Direction: values.Get(key),
			}

			continue
		}

		// _filter:column:operator:optFilterGroup=value,values
		if strings.HasPrefix(key, filter) {
			parts := strings.Split(key, ":")
			if len(parts) < 3 {
				continue
			}
			column := parts[1]
			operator := parts[2]
			index := 0

			if len(parts) >= 4 {
				index = intVal(parts[3])
			}

			dto = extendFilter(dto, index)

			dto.Filter[index] = append(dto.Filter[index], Filter{
				Column:   column,
				Operator: operator,
				Value:    strings.Split(values.Get(key), ","),
			})
		}
	}

	return dto
}

func intVal(value string) int {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return val
}

func extendSorter(dto GridDto, length int) GridDto {
	if len(dto.Sorter) > length {
		return dto
	}

	old := dto.Sorter

	dto.Sorter = make([]Sorter, length+1)
	for index, sorter := range old {
		dto.Sorter[index] = sorter
	}

	return dto
}

func extendFilter(dto GridDto, length int) GridDto {
	if len(dto.Filter) > length {
		return dto
	}

	old := dto.Filter
	dto.Filter = make([][]Filter, length+1)
	for index, filter := range old {
		dto.Filter[index] = filter
	}

	return dto
}
