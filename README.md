# REST API - data grid

## Filtrování
Filter podmínka je splněna, pokud se všechny prvky v poli filter vyhodnotí jako *TRUE* (je na ně aplikován *AND*).

Jednotlivé podmínky filtru jsou splněny, pokud alespoň jedna z nich je *TRUE* (aplikuje se *OR*).

Dostupné operátory (field `operator`) filtru:

*EQ*, *NEQ*, *GT*, *LT*, *GTE*, *LTE*, *LIKE*, *STARTS*, *ENDS*, *EMPTY*, *NEMPTY*, *BETWEEN*, *NBETWEEN*

## Řazení
Je možné sorter poslat jako null a v tom případě se server rozhodne o defaultním řazení. V případě cursoringu se nemusí posílat vůbec (server ho bude ignorovat).

Field direction může být: *ASC*, *DESC*

## Stránkování
Stránkovat lze klasicky s offsetem pomocí paging, nebo optimálněji bez offsetu, přes nějaké unikátní, řazené id, pomocí cursoring.

## Chyby
Pokud API nebude podporovat operátor, stránkování, atp. vrátí status *422 UNPROCESSABLE ENTITY*.

## Příklad
Následující příklad filtru je ekvivalent k

`(guid = "some-guid" || guid != "some-guid") && search = "search"`

Response - paging
```json
{
 "filter": [
   [
     {
       "column": "guid",
       "operator": "EQ",
       "value": [
         "some-guid"
       ]
     },
	 {
       "column": "guid",
       "operator": "NEQ",
       "value": [
         "some-guid-1", "some-guid-2"
       ]
     }
   ]
 ],
 "search": "value",
 "sorter": [
   {
     "column": "firstname",
     "direction": "ASC"
   },
   {
     "column": "created",
     "direction": "DESC"
   }
 ],
 "paging": {
   "page": 1,
   "itemsPerPage": 20
 }
}
```
Response
```json
{
  "items": [
    {
      "guid": "some-guid"
    }
  ],
  "paging": {
    "page": 1,
    "total": 0,
    "itemsPerPage": 20,
    "lastPage": 1,
    "nextPage": 1,
    "previousPage": 1
  },
  "search": "value",
  "sorter": [
    {
     "column": "firstname",
     "direction": "ASC"
    },
    {
      "column": "created",
      "direction": "DESC"
    }
  ],
  "filter": [
    [
      {
        "operator": "EQ",
        "column": "guid",
        "value": [
          "some-guid"
        ]
      }
    ],
    [
      {
        "operator": "FL",
        "column": "sapId",
        "value": null
      }
    ]
  ]
}
```
Request - cursoring
```json
{
 "filter": [
   [
     {
       "column": "guid",
       "operator": "EQ",
       "value": [
         "some-guid"
       ]
     }
   ]
 ],
 "cursoring": {
   "lastId": 0,
   "itemsPerPage": 20
 }
}
```
Response
```json
{
  "items": [
    {
      "guid": "some-guid"
    }
  ],
  "filter": [
    [
      {
        "operator": "EQ",
        "column": "guid",
        "value": [
          "some-guid"
        ]
      }
    ]
  ],
  "cursoring": {
    "lastId": 19,
    "itemsPerPage": 20
  }
}
```