package filter

import (
	"time"

	"github.com/google/uuid"
)

var fittedNestedModel = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: search{value: "value", filters: [][]filter{
		{{column: "x.name", Operator: "LIKE", values: []interface{}{"value"}}},
	}},
	sorter: sorter{
		{Column: "firstName", Direction: "ASC"},
		{Column: "created", Direction: "DESC"},
		{Column: "x.id", column: "x.id"},
	},
	filter: [][]filter{
		{
			{Column: "x.id", column: "x.id", Operator: "EQ",
				Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"},
				values: []interface{}{UUID{UUID: uuid.MustParse("f1611454-debb-4d9f-bd78-83f0d38b0176")}},
			},
			{Column: "x.id", column: "x.id", Operator: "NEQ",
				Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"},
				values: []interface{}{UUID{UUID: uuid.MustParse("853649c7-9ff9-4572-b5b2-98f8da30e20a")}, UUID{UUID: uuid.MustParse("4b27dc87-e969-4bc3-afc5-195403fea580")}},
			},
		}, {
			{Column: "x.value", column: "x.value", Operator: "GT", Values: []interface{}{10.0}, values: []interface{}{10.0}},
			{Column: "x.value", column: "x.value", Operator: "LTE", Values: []interface{}{10.0}, values: []interface{}{10.0}},
		}, {
			{Column: "x.value", column: "x.value", Operator: "LT", Values: []interface{}{"10"}, values: []interface{}{"10"}},
			{Column: "x.value", column: "x.value", Operator: "GTE", Values: []interface{}{"10"}, values: []interface{}{"10"}},
		}, {
			{Column: "x.name", column: "x.name", Operator: "LIKE", Values: []interface{}{"John Smith"}, values: []interface{}{"John Smith"}},
		}, {
			{Column: "x.name", column: "x.name", Operator: "STARTS", Values: []interface{}{"John"}, values: []interface{}{"John"}},
			{Column: "x.name", column: "x.name", Operator: "ENDS", Values: []interface{}{"Smith"}, values: []interface{}{"Smith"}},
		}, {
			{Column: "s.activatedAt", column: "s.activated_at",
				Values: []interface{}{"2002-10-02T15:00:00Z"},
				values: []interface{}{time.Date(2002, 10, 2, 15, 00, 00, 0, time.UTC)}},
			{Column: "s.activatedAt", column: "s.activated_at", Operator: "EMPTY"},
			{Column: "s.activatedAt", column: "s.activated_at", Operator: "NEMPTY"},
		}, {
			{Column: "s.createdAt", column: "s.created_at", Operator: "BETWEEN",
				Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"},
				values: []interface{}{
					time.Date(2000, 10, 2, 15, 00, 00, 0, time.UTC),
					time.Date(2020, 10, 2, 15, 00, 00, 0, time.UTC),
				}},
			{Column: "s.createdAt", column: "s.created_at", Operator: "NBETWEEN",
				Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"},
				values: []interface{}{
					time.Date(2000, 10, 2, 15, 00, 00, 0, time.UTC),
					time.Date(2020, 10, 2, 15, 00, 00, 0, time.UTC),
				}},
		},
		{
			{Column: "x.id", column: "x.id", Operator: "IN",
				Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"},
				values: []interface{}{UUID{UUID: uuid.MustParse("f1611454-debb-4d9f-bd78-83f0d38b0176")}},
			},
			{Column: "x.id", column: "x.id", Operator: "NIN",
				Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"},
				values: []interface{}{UUID{UUID: uuid.MustParse("853649c7-9ff9-4572-b5b2-98f8da30e20a")}, UUID{UUID: uuid.MustParse("4b27dc87-e969-4bc3-afc5-195403fea580")}},
			},
		},
	},
}
