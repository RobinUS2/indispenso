package data_table

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptySearchValue(t *testing.T) {
	ds := &DefaultStore{}
	row := newTableRow(map[string]interface{}{"col1": "test1", "col2": "invalid"})

	filter := &SearchDefinition{Value: "", Regexp: false}
	assert.True(t, ds.FilterRow(row, filter, &SortableRowSet{}))

	rowSet := &SortableRowSet{cols: []*ColumnDefinition{&ColumnDefinition{Name: "col1", Searchable: true, Search: &SearchDefinition{Value: "", Regexp: false}}}}
	assert.True(t, ds.FilterRow(row, filter, rowSet))
}

func TestSearchForValue(t *testing.T) {
	ds := &DefaultStore{}
	row := newTableRow(map[string]interface{}{"col1": "test1", "col2": "invalid"})

	filter := &SearchDefinition{Value: "testable", Regexp: false}
	rowSet := &SortableRowSet{}
	assert.False(t, ds.FilterRow(row, filter, rowSet))

	filter = &SearchDefinition{Value: "test1", Regexp: false}
	assert.True(t, ds.FilterRow(row, filter, rowSet))

	filter = &SearchDefinition{Value: "", Regexp: false}
	rowSet = &SortableRowSet{cols: []*ColumnDefinition{&ColumnDefinition{Name: "col1", Searchable: true, Search: &SearchDefinition{Value: "testable", Regexp: false}}}}
	assert.False(t, ds.FilterRow(row, filter, rowSet))

	filter = &SearchDefinition{Value: "", Regexp: false}
	rowSet = &SortableRowSet{cols: []*ColumnDefinition{&ColumnDefinition{Name: "col1", Searchable: true, Search: &SearchDefinition{Value: "test1", Regexp: false}}}}
	assert.True(t, ds.FilterRow(row, filter, rowSet))

	filter = &SearchDefinition{Value: "testable", Regexp: false}
	rowSet = &SortableRowSet{cols: []*ColumnDefinition{&ColumnDefinition{Name: "col1", Searchable: true, Search: &SearchDefinition{Value: "test1", Regexp: false}}}}
	assert.False(t, ds.FilterRow(row, filter, rowSet))

	filter = &SearchDefinition{Value: "testable", Regexp: false}
	rowSet = &SortableRowSet{cols: []*ColumnDefinition{&ColumnDefinition{Name: "col1", Searchable: true, Search: &SearchDefinition{Value: "testable", Regexp: false}}}}
	assert.False(t, ds.FilterRow(row, filter, rowSet))

}
