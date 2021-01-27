package filter

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

const (
	filterable = "filter"
	sortable   = "sort"
	searchable = "search"
	skip       = "skip"

	Empty    = "EMPTY"
	Nempty   = "NEMPTY"
	Like     = "LIKE"
	Nlike    = "NLIKE"
	Eq       = "EQ"
	Neq      = "NEQ"
	Between  = "BETWEEN"
	Nbetween = "NBETWEEN"
	Gt       = "GT"
	Lt       = "LT"
	Gte      = "GTE"
	Lte      = "LTE"
	In       = "IN"
	Nin      = "NIN"
	Starts   = "STARTS"
	Ends     = "ENDS"
)

type FilterCallback func(field, operator string, values []string) squirrel.Sqlizer

type QueryCallback func(qb squirrel.SelectBuilder, field, operator string, values []string) squirrel.SelectBuilder

type callbackStack []callbackStackItem

type callbackStackItem struct {
	callback QueryCallback
	field    string
	operator string
	values   []string
}

type Grid interface {
	SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder
}

/// Callbacks defining filter condition in place
type FilterCallbacks interface {
	FilterCallbacks() map[string]FilterCallback
}

/// Global callbacks on top of finished queryBuilder (use for HAVING filter)
type QueryCallbacks interface {
	QueryCallbacks() map[string]QueryCallback
}

func GetData(model Grid, dto GridDto, db *sqlx.DB, resultSet interface{}) (GridDto, error) {
	if dto.Paging.Size <= 0 {
		dto.Paging.Size = defaultSize
	}
	if dto.Paging.Page <= 0 {
		dto.Paging.Page = 1
	}

	filterCalls := map[string]FilterCallback{}
	fcl, ok := interface{}(model).(FilterCallbacks)
	if ok {
		filterCalls = fcl.FilterCallbacks()
	}

	queryCalls := map[string]QueryCallback{}
	qcl, ok := interface{}(model).(QueryCallbacks)
	if ok {
		queryCalls = qcl.QueryCallbacks()
	}

	callbacks := callbackStack{}

	// Filters
	andQueries := squirrel.And{}
	for _, filters := range dto.Filter {
		var orQeuries squirrel.Or
		for _, filter := range filters {
			if hasTag(model, filter.Column, filterable) {
				tagName := taggedName(model, filter.Column)
				if callback, ok := filterCalls[tagName]; ok {
					orQeuries = append(orQeuries, callback(tagName, filter.Operator, filter.Value))
				} else if callback, ok := queryCalls[tagName]; ok {
					callbacks = append(callbacks, callbackStackItem{
						callback: callback,
						field:    tagName,
						operator: filter.Operator,
						values:   filter.Value,
					})
				} else {
					orQeuries = append(orQeuries, FormQuery(tagName, filter.Operator, filter.Value, true))
				}
			} else {
				return dto, fmt.Errorf("field [%s] is not tagged for filtering", filter.Column)
			}
		}

		if orQeuries != nil {
			andQueries = append(andQueries, orQeuries)
		}
	}

	// Search
	if dto.Search != "" {
		fields := getSearchFields(model)
		var orQueries squirrel.Or
		for _, field := range fields {
			orQueries = append(orQueries, FormQuery(taggedName(model, field), Like, []string{dto.Search}, true))
		}
		if orQueries != nil {
			andQueries = append(andQueries, orQueries)
		}
	}

	sql, args, err := andQueries.ToSql()
	if err != nil {
		return dto, err
	}

	qb := model.SearchQuery(squirrel.Select("*")).Where(sql, args...)
	sqlC, argsC, err := callbacks.merge(qb).ToSql()
	if err != nil {
		return dto, err
	}

	// Remove * should user define custom Column for selecting (duplicate fields)
	sqlC = strings.Replace(sqlC, "SELECT *,", "SELECT ", -1)

	// Count query
	position := 1
	words := strings.Split(sqlC, " ")
	endAt := 0
	// Finds matching FROM to first SELECT
	for i, word := range words[1:] {
		stripped := strings.Trim(word, "()")
		if strings.HasPrefix(stripped, "SELECT") {
			position++
		} else if strings.HasPrefix(stripped, "FROM") {
			position--
			if position == 0 {
				endAt = i + 1
				break
			}
		}
	}

	// If joining tables -> replace * to avoid duplicate
	position = 0
	for i, word := range words[endAt+1:] {
		if strings.Contains(word, "(") {
			position++
		}
		if strings.Contains(word, ")") {
			position--
		} else if word == "JOIN" && position == 0 {
			alias := words[endAt+i]
			if alias == "LEFT" || alias == "RIGHT" || alias == "INNER" || alias == "OUTER" {
				alias = words[endAt+i-1]
			}
			sqlC = strings.Replace(sqlC, "SELECT * FROM", fmt.Sprintf("SELECT %s.* FROM", alias), 1)

			break
		}
	}

	// Everything in between first SELECT and it's matching FROM
	sqlInnerSelect := strings.Join(words[1:endAt], " ")

	if !strings.Contains(sqlC, " HAVING ") {
		sqlC = "SELECT COUNT(*)"
		sqlC = fmt.Sprintf("SELECT COUNT(*) %s", strings.Join(words[endAt:], " "))

		// Remove GROUP BY from Count query and use it's value in COUNT distinction
		reg := regexp.MustCompile(`(GROUP BY ([^ ]+))`)
		matches := reg.FindAllStringSubmatch(sqlC, -1)
		if len(matches) == 1 && len(matches[0]) == 3 {
			sqlC = strings.Replace(sqlC, matches[0][1], "", 1)
			sqlC = strings.Replace(sqlC, "COUNT(*)", fmt.Sprintf("COUNT(DISTINCT %s)", matches[0][2]), 1)
		}
	} else {
		// Having can't be counted in standart way -> wrap the whole sql expression with another SELECT COUNT
		sqlC = fmt.Sprintf("SELECT COUNT(*) FROM (%s) counter_alias;", sqlC)
	}

	var count []int
	err = db.Select(&count, sqlC, argsC...)
	if err != nil {
		return dto, err
	}

	// Count query ends there

	qb = createSelects(model, sqlInnerSelect).Where(sql, args...)
	qb = callbacks.merge(qb)

	// OrderBy
	for _, sorter := range dto.Sorter {
		if hasTag(model, sorter.Column, sortable) {
			qb = qb.OrderBy(fmt.Sprintf("%s %s", Name(taggedName(model, sorter.Column), true), sorter.Direction))
		} else {
			return dto, fmt.Errorf("field [%s] is not tagged for sorting", sorter.Column)
		}
	}

	// Paging
	qb = qb.Limit(uint64(dto.Paging.Size)).
		Offset(uint64((dto.Paging.Page - 1) * dto.Paging.Size))

	sql, args, err = qb.ToSql()
	if err != nil {
		return dto, err
	}

	c := count[0]
	last := c / dto.Paging.Size
	if last*dto.Paging.Size < c {
		last++
	}

	if last <= 0 {
		last = 1
	}

	dto.Paging.Total = c
	dto.Paging.LastPage = last

	dto.Paging.PreviousPage = dto.Paging.Page - 1
	if dto.Paging.PreviousPage <= 0 {
		dto.Paging.PreviousPage = 1
	}

	dto.Paging.NextPage = dto.Paging.Page + 1
	if dto.Paging.NextPage > last {
		dto.Paging.NextPage = last
	}

	if err = db.Select(resultSet, sql, args...); err != nil {
		return dto, err
	}

	dto.Items = resultSet

	return dto, nil
}

func FormQuery(field, operator string, values []string, safe bool) squirrel.Sqlizer {
	return squirrel.Expr(
		OperatorToQuery(operator, field, len(values), safe),
		ParseValues(values, operator)...,
	)
}

func OperatorToQuery(operator, column string, values int, safe bool) string {
	switch operator {
	case Eq:
		return fmt.Sprintf("%s = ?", Name(column, safe))
	case Neq:
		return fmt.Sprintf("%s != ?", Name(column, safe))
	case Empty:
		return fmt.Sprintf("%s IS NULL", Name(column, safe))
	case Nempty:
		return fmt.Sprintf("%s IS NOT NULL", Name(column, safe))
	case In:
		vals := make([]string, values)
		for i := 0; i < values; i++ {
			vals[i] = "?"
		}
		return fmt.Sprintf("%s IN (%s)", Name(column, safe), strings.Join(vals, ","))
	case Nin:
		vals := make([]string, values)
		for i := 0; i < values; i++ {
			vals[i] = "?"
		}
		return fmt.Sprintf("%s NOT IN (%s)", Name(column, safe), strings.Join(vals, ","))
	case Like:
		return fmt.Sprintf("%s LIKE ?", Name(column, safe))
	case Starts:
		return fmt.Sprintf("%s LIKE ?", Name(column, safe))
	case Ends:
		return fmt.Sprintf("%s LIKE ?", Name(column, safe))
	case Nlike:
		return fmt.Sprintf("%s NOT LIKE ?", Name(column, safe))
	case Gt:
		return fmt.Sprintf("%s > ?", Name(column, safe))
	case Lt:
		return fmt.Sprintf("%s < ?", Name(column, safe))
	case Lte:
		return fmt.Sprintf("%s <= ?", Name(column, safe))
	case Gte:
		return fmt.Sprintf("%s >= ?", Name(column, safe))
	case Between:
		return fmt.Sprintf("%s BETWEEN ? AND ?", Name(column, safe))
	case Nbetween:
		return fmt.Sprintf("%s NOT BETWEEN ? AND ?", Name(column, safe))
	}

	return fmt.Sprintf("%s = ?", Name(column, safe))
}

func Name(name string, safe bool) string {
	if safe {
		cols := strings.Split(name, ".")
		for i, col := range cols {
			cols[i] = fmt.Sprintf("`%s`", col)
		}

		return strings.Join(cols, ".")
	}

	return name
}

func ParseValues(values []string, operator string) []interface{} {
	vals := make([]interface{}, len(values))
	for key, val := range values {
		switch operator {
		case Like:
			vals[key] = fmt.Sprintf("%%%s%%", val)
		case Nlike:
			vals[key] = fmt.Sprintf("%%%s%%", val)
		case Starts:
			vals[key] = fmt.Sprintf("%s%%", val)
		case Ends:
			vals[key] = fmt.Sprintf("%%%s", val)
		case Empty, Nempty:
			return nil
		default:
			vals[key] = val
		}
	}

	return vals
}

func createSelects(model Grid, selects string) squirrel.SelectBuilder {
	var listed []string
	for _, field := range strings.Split(selects, ",") {
		field = strings.TrimSpace(field)
		if len(field) > 0 {
			parts := strings.Split(field, " ")
			if len(parts) > 1 {
				listed = append(listed, parts[len(parts)-1])
			}
		}
	}

	fType := reflect.TypeOf(model)
	count := fType.NumField()
	var fields []string

	for i := 0; i < count; i++ {
		fieldName := fType.Field(i).Tag.Get("db")

		if !hasTag(model, fType.Field(i).Name, skip) {
			if fieldName == "" {
				fieldName = lowerFirst(fType.Field(i).Name)
			}

			ok := true
			for _, inList := range listed {
				if inList == fieldName {
					ok = false
					break
				}
			}

			if ok {
				fields = append(fields, fmt.Sprintf("%s as `%s`", Name(fieldName, true), fieldName))
			}
		}
	}

	return model.SearchQuery(squirrel.Select(fields...))
}

func taggedName(model Grid, column string) string {
	field, ok := reflect.TypeOf(model).FieldByName(strings.Title(column))
	if !ok {
		return column
	}

	name := field.Tag.Get("db")
	if name != "" {
		return name
	}

	return column
}

func hasTag(model Grid, column, operation string) bool {
	field, ok := reflect.TypeOf(model).FieldByName(strings.Title(column))
	if !ok {
		return false
	}

	for _, tag := range strings.Split(field.Tag.Get("grid"), ",") {
		if strings.Trim(tag, " ") == operation {
			return true
		}
	}

	return false
}

func getSearchFields(model Grid) []string {
	var fields []string
	fType := reflect.TypeOf(model)
	count := fType.NumField()
	for i := 0; i < count; i++ {
		name := fType.Field(i).Tag.Get("json")
		if name == "" {
			name = fType.Field(i).Name
		}

		if hasTag(model, name, searchable) {
			fields = append(fields, name)
		}
	}

	return fields
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}

	r, n := utf8.DecodeRuneInString(s)

	return string(unicode.ToLower(r)) + s[n:]
}

func (qm callbackStack) merge(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
	for _, stack := range qm {
		qb = stack.callback(qb, stack.field, stack.operator, stack.values)
	}

	return qb
}
