package filter

var filterParsed = Filter{
	paging: paging{
		Page:         1,
		ItemsPerPage: 20,
	},
	search: search{value: "value"},
	sorter: sorter{
		{Column: "firstName", Direction: "ASC"},
		{Column: "created", Direction: "DESC"},
		{Column: "id"},
	},
	filter: [][]filter{
		{
			{Column: "id", Operator: "EQ", Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"}},
			{Column: "id", Operator: "NEQ", Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"}},
		}, {
			{Column: "value", Operator: "GT", Values: []interface{}{10.0}},
			{Column: "value", Operator: "LTE", Values: []interface{}{10.0}},
		}, {
			{Column: "value", Operator: "LT", Values: []interface{}{"10"}},
			{Column: "value", Operator: "GTE", Values: []interface{}{"10"}},
		}, {
			{Column: "name", Operator: "LIKE", Values: []interface{}{"John Smith"}},
		}, {
			{Column: "name", Operator: "STARTS", Values: []interface{}{"John"}},
			{Column: "name", Operator: "ENDS", Values: []interface{}{"Smith"}},
		}, {
			{Column: "activatedAt", Values: []interface{}{"2002-10-02T15:00:00Z"}},
			{Column: "activatedAt", Operator: "EMPTY"},
			{Column: "activatedAt", Operator: "NEMPTY"},
		}, {
			{Column: "createdAt", Operator: "BETWEEN", Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
			{Column: "createdAt", Operator: "NBETWEEN", Values: []interface{}{"2000-10-02T15:00:00Z", "2020-10-02T15:00:00Z"}},
		}, {
			{Column: "id", Operator: "IN", Values: []interface{}{"f1611454-debb-4d9f-bd78-83f0d38b0176"}},
			{Column: "id", Operator: "NIN", Values: []interface{}{"853649c7-9ff9-4572-b5b2-98f8da30e20a", "4b27dc87-e969-4bc3-afc5-195403fea580"}},
		},
	},
}
