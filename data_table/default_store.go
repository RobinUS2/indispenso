package data_table

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cast"
	"net/http"
)

type DefaultStore struct {
	rows       RowSet
	dataSource func(*DefaultStore) *DefaultStore
	search     *SearchDefinition
	filtered   int
	total      int
}

func newDefaultStore(queryFunc func(*DefaultStore) *DefaultStore) *DefaultStore {
	return &DefaultStore{dataSource: queryFunc}
}

func (ds *DefaultStore) AddRow(data *TableRow) error {
	if ds.rows == nil {
		return errors.New("Missconfigured default store need to setup rowset")
	}

	if ds.FilterRow(data, ds.search, ds.rows) {
		ds.rows.Add(data)
		ds.filtered++
	}
	ds.total++
	return nil
}

func (ds *DefaultStore) GetData() RowSet {
	return ds.rows
}

func (ds *DefaultStore) QueryData(sr *SearchRequest) *TableResult {
	ds.rows = newSortableRowSet(uint(sr.Start), uint(sr.End()), sr.Columns)
	ds.search = sr.Search
	ds.filtered = 0
	ds.total = 0

	result := ds.dataSource(ds)

	return newTableResult(sr, result.GetData(), result.total, result.filtered)
}

func (ds *DefaultStore) FilterRow(row *TableRow, filter *SearchDefinition, rowSet RowSet) bool {
	if row == nil || len(row.Data) == 0 {
		return false
	}
	foundGlobally := true
	for name, value := range row.Data {
		strValue := cast.ToString(value)

		if filter.IsApplicable() {
			if filter.IsMatch(strValue) {
				return true
			} else {
				foundGlobally = false
			}
		}

		if col := rowSet.ColumnByName(name); col != nil {
			if col.Searchable && col.Search.IsApplicable() {
				if !col.Search.IsMatch(strValue) {
					return false
				}
			}
		}
	}

	return foundGlobally
}

func (ds *DefaultStore) CreateRow(data map[string]interface{}) *TableRow {
	return newTableRow(data)
}

func DefaultStoreHandler(queryFunc func(*DefaultStore) *DefaultStore) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return DataStoreHandler(newDefaultStore(queryFunc))
}
