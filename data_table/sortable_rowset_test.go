package data_table

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddRow(t *testing.T) {
	rowSet := newSortableRowSet(0, 10, []*ColumnDefinition{&ColumnDefinition{Name: "test"}})

	row := &TableRow{Data: map[string]interface{}{"test": "testData"}}

	err := rowSet.Add(row)
	assert.NoError(t, err)

	assert.Equal(t, uint(1), rowSet.Rows.Len())
	rows := rowSet.GetRows()
	assert.Equal(t, rows[0], row)
}

func TestAddMultiRow(t *testing.T) {
	rowSet := newSortableRowSet(0, 10, []*ColumnDefinition{&ColumnDefinition{Name: "test"}})

	row1 := &TableRow{Data: map[string]interface{}{"test": "testData1"}}
	row2 := &TableRow{Data: map[string]interface{}{"test": "testData2"}}

	err := rowSet.Add(row1)
	assert.NoError(t, err)
	err = rowSet.Add(row2)
	assert.NoError(t, err)

	assert.Equal(t, uint(2), rowSet.Rows.Len())
	rows := rowSet.GetRows()
	assert.Contains(t, rows, row1)
	assert.Contains(t, rows, row2)
}

func TestIsLessOrdering(t *testing.T) {
	rowSet := newSortableRowSet(0, 10, []*ColumnDefinition{&ColumnDefinition{Name: "test", Orderable: true, Order: ORDER_DESC}})

	row1 := &TableRow{Data: map[string]interface{}{"test": "testData1"}}
	row2 := &TableRow{Data: map[string]interface{}{"test": "testData2"}}
	row3 := &TableRow{Data: map[string]interface{}{"test": "testData3"}}
	row10 := &TableRow{Data: map[string]interface{}{"test": "testData10"}}

	assert.False(t, rowSet.IsLess(row2, row1))
	assert.True(t, rowSet.IsLess(row1, row2))
	assert.True(t, rowSet.IsLess(row1, row3))
	assert.False(t, rowSet.IsLess(row3, row3))
	assert.False(t, rowSet.IsLess(row10, row1))
}

func TestIsLessOrderingAscending(t *testing.T) {
	rowSet := newSortableRowSet(0, 10, []*ColumnDefinition{&ColumnDefinition{Name: "test", Orderable: true, Order: ORDER_ASC}})

	row1 := &TableRow{Data: map[string]interface{}{"test": "testData1"}}
	row2 := &TableRow{Data: map[string]interface{}{"test": "testData2"}}
	row3 := &TableRow{Data: map[string]interface{}{"test": "testData3"}}
	row10 := &TableRow{Data: map[string]interface{}{"test": "testData10"}}

	assert.True(t, rowSet.IsLess(row2, row1))
	assert.False(t, rowSet.IsLess(row1, row2))
	assert.False(t, rowSet.IsLess(row1, row3))
	assert.False(t, rowSet.IsLess(row3, row3))
	assert.True(t, rowSet.IsLess(row10, row1))
}

func TestNotSortedSetIsLessConsistentResult(t *testing.T) {
	rowSet := newSortableRowSet(0, 10, []*ColumnDefinition{&ColumnDefinition{Name: "test", Orderable: false}})

	row1 := &TableRow{Data: map[string]interface{}{"test": "testData1"}}
	row100 := &TableRow{Data: map[string]interface{}{"test": "testData100"}}

	assert.Equal(t, rowSet.IsLess(row1, row100), !rowSet.IsLess(row100, row1))
	assert.False(t, rowSet.IsLess(row1, row1))
}
