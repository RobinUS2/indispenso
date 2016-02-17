package data_table

import (
	"encoding/json"
	"fmt"
	"github.com/HuKeping/rbtree"
	"github.com/spf13/cast"
	"strings"
)

type SortableRowSet struct {
	Limit uint
	Start uint
	cols  []*ColumnDefinition
	Rows  *rbtree.Rbtree
}

func newSortableRowSet(start uint, limit uint, cols []*ColumnDefinition) *SortableRowSet {
	return &SortableRowSet{Limit: limit, Start: start, cols: cols, Rows: rbtree.New()}
}

func (rs *SortableRowSet) ColumnByName(name string) *ColumnDefinition {
	for _, col := range rs.cols {
		if col.Name == name {
			return col
		}
	}
	return nil
}

func (rs *SortableRowSet) validateRow(row *TableRow) bool {
	if len(rs.cols) != len(row.Data) {
		return false
	}

	for _, colDef := range rs.cols {
		if _, ok := row.Data[colDef.Name]; !ok {
			return false
		}
	}

	return true
}

func (rs *SortableRowSet) IsLess(origin *TableRow, than *TableRow) bool {
	for _, val := range rs.cols {
		if val.Orderable && val.Order != ORDER_UNDEFINED {
			if res := strings.Compare(cast.ToString(origin.Data[val.Name]), cast.ToString(than.Data[val.Name])); res != 0 {
				if val.Order == ORDER_DESC {
					return res == -1
				} else {
					return res == 1
				}
			}
		}
	}
	//no ordering provided, base on string pointers
	return strings.Compare(fmt.Sprintf("%p", origin.Data), fmt.Sprintf("%p", than.Data)) == -1
}

func (rs *SortableRowSet) Add(row *TableRow) error {
	if !(rs.Rows.Len() > rs.Limit && rs.Rows.Min().Less(row)) {
		row.IsLess = rs.IsLess
		rs.Rows.Insert(row)
	}
	return nil
}

func (rs *SortableRowSet) GetRows() []*TableRow {
	res := []*TableRow{}
	if rs.Rows.Len() >= rs.Start {
		count := uint(0)

		rs.Rows.Descend(rs.Rows.Max(), func(row rbtree.Item) bool {
			if count >= rs.Start && count < rs.Limit {
				tblRow := row.(*TableRow)
				res = append(res, tblRow)
			}
			count++
			return true
		})
	}
	return res
}

func (rs *SortableRowSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(rs.GetRows())
}
