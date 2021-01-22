# GoFilter

## Request format

Filter accepts GET requests with all parameters included within query params:

- `_page=1` requested page
- `_size=10` items per page
- `_search=wordToSearch` full text search on marked fields
- `_filter:column:operator:optFilterGroup=value,value2,value3` read below
- `_sorter:column:optIndex=direction` read below

##### Allowed filter operators:
- no-value: `EMPTY`, `NEMPTY` (send anything into query param value: bool, single char, ...) - checks for NULL values
- single-valued: `EQ`, `NEQ`, `GT`, `GTE`, `LT`, `LTE`, `LIKE`, `NLIKE`, `STARTS`, `ENDS`
- multi-valued: `BETWEEN`, `NBETWEEN`, `IN`, `NIN`

##### Filter group

To distinguish between AND and OR conditions, grid uses FilterGroup.
- Filters within single group are joined with OR.
- Groups are joined with AND.
- Unspecified are treated as new group
- Search is separate filter group

##### Examples:
```
WHERE (name = 'namae') AND (surname = 'surnamae') 

_filter:name:EQ=namae & _filter:surname:EQ=surnamae
_filter:name:EQ:1=namae & _filter:surname:EQ:2=surnamae
```
```
WHERE (name = 'namae' OR surname = 'surnamae') 

_filter:name:EQ:1=namae & _filter:surname:EQ:1=surnamae
```
```
WHERE (name = 'namae' OR surname = 'surnamae') AND (present IS NOT NULL)

_filter:name:EQ:1=namae & _filter:surname:EQ:1=surnamae & _filter:present:NEMPTY:2=true
```
```
WHERE (surname LIKE 'sur%') AND (name LIKE '%nae%' OR middlename LIKE '%nae%') ORDER BY id ASC LIMIT 10, 20

_filter:surname:STARTS=sur & _search:=nae & _sorter:id=ASC & _page=3 & _size=10
```

##### Sorter

- optIndex: use numeric values 1..N to specify order of ORDER BY clauses
- direction: `ASC`, `DESC`

## Implementation

Grid if defined within struct(entity)'s tags under `grid` key
Available options:
 - `filter` marks field as filterable -> if not marked grid throws an error when filtered
 - `sort` marks field as sortable -> if not marked grid throws an error when sorted
 - `search` includes field in fulltext search
 - `skip` excludes field from grid selects

Each struct MUST implement filter.Grid interface

Request must be parsed into filter.GridDto struct which is used to return result as well
Dto can be created from http.Request with `filter.CreateGridDto` method

## Basic usage
```go
type Entity struct {
    Id   int    `grid:"filter,sort"`
    Name string `grid:"filter,sort,search"`
}

func (e Entity) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
    return qb.From("entity")
}

func filter(request http.Request) (filter.GridDto, error) {
    dto := filter.CreateGridDto(&request)

    var res []Entity
 
    return filter.GetData(Entity{}, dto, MariaDB, &res)
 }
```

## Join tables

Joined tables are specified within SearchQuery method. All fields must be aliased with `db` tag to specify correct mappings.

```go
type Entity struct {
    Id      int    `db:"e.id"     grid:"filter,sort"`
    Name    string `db:"e.name"   grid:"filter,sort,search"`
    TagId   int    `db:"e.tag_id" grid:"filter"`
    TagName string `db:"t.name"   grid:"filter"`
}

func (e Entity) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
    return qb.
        From("entity e").
        LeftJoin("tag t ON e.tag_id = t.id")
}
```

### OneToMany joins

To get correct number of rows, OneToMany relations must by grouped within SearchQuery. For filtering on joined fields, define them in entityStruct and mark as `skip`. Should the joined data be needed, fetch them in separate query after grid filtering (mark target field as `skip` as well).

```go
type Entity struct {
    Id      int    `db:"e.id"   grid:"filter,sort"`
    Name    string `db:"e.name" grid:"filter,sort,search"`
    // Target field for fetch Tags later
    Tags    []Tag  `grid:"skip"`
    // Fake field to allow filtering
    TagName   int  `db:"t.name" grid:"filter,skip"`
}

func (e Entity) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
    return qb.
        From("entity e").
        LeftJoin("tag t ON e.id = t.entity_id").
        GroupBy("e.id")
}
```

## Custom callbacks

### Filter callbacks

Should you require filter overrides (like filtering on COUNT-ed field) define a FilterCallback. These are unlike QueryCallbacks applied only within WHERE clause on specific place defined in filter request

```go
type Entity struct {
    Id       int    `db:"e.id"     grid:"filter,sort"`
    Name     string `db:"e.name"   grid:"filter,sort,search"`
    TagCount int    `db:"tagCount" grid:"filter"`
}

func (e Entity) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
    return qb.
        Column("(SELECT COUNT(id) FROM tag WHERE entity_id = e.id) as tagCount").
        From("entity e").
        LeftJoin("tag t ON e.id = t.entity_id").
        GroupBy("e.id")
}

func (e Entity) FilterCallbacks() map[string]FilterCallback {
    return map[string]FilterCallback{
        "tagCount": func(field, operator string, values []string) squirrel.Sqlizer {
            return FormQuery("COUNT(t.id) as tagCount", operator, values, false)
        },
    }
}
```

### Query callbacks

Used for extra queries like HAVING clause. Unlike Filter callback, those are called after QueryBuilder is formed.

```go
type Entity struct {
    Id       int    `db:"e.id"     grid:"filter,sort"`
    Name     string `db:"e.name"   grid:"filter,sort,search"`
    TagCount int    `db:"tagCount" grid:"filter"`
}

func (e Entity) SearchQuery(qb squirrel.SelectBuilder) squirrel.SelectBuilder {
    return qb.
        Column("(SELECT COUNT(id) FROM tag WHERE entity_id = e.id) as tagCount").
        From("entity e").
        LeftJoin("tag t ON e.id = t.entity_id").
        GroupBy("e.id")
}

func (T TGrid) QueryCallbacks() map[string]QueryCallback {
    return map[string]QueryCallback{
        "tagCount": func(qb squirrel.SelectBuilder, field, operator string, values []string) squirrel.SelectBuilder {
            return qb.Having(FormQuery("tagCount", operator, values, false))
        },
    }
}
```
