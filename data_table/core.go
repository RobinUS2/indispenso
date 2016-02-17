package data_table

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HuKeping/rbtree"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cast"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type DataTableOrder string

const (
	ORDER_DESC      = DataTableOrder("desc")
	ORDER_ASC       = DataTableOrder("asc")
	ORDER_UNDEFINED = DataTableOrder("")
)

type TableStore interface {
	QueryData(*SearchRequest) *TableResult
}

type RowSet interface {
	GetRows() []*TableRow
	ColumnByName(string) *ColumnDefinition
	Add(*TableRow) error
}

type TableResult struct {
	Draw       int                 `json:"draw"`
	Total      int                 `json:"recordsTotal"`
	Filtered   int                 `json:"recordsFiltered"`
	Data       []*TableRow         `json:"data"`
	definition []*ColumnDefinition `json:"-"`
}

func newTableResult(searchRequest *SearchRequest, rowSet RowSet, total int, filtered int) *TableResult {
	return &TableResult{definition: searchRequest.Columns, Draw: searchRequest.Draw, Data: rowSet.GetRows(), Total: total, Filtered: filtered}
}

type TableRow struct {
	Data     map[string]interface{}
	RowId    string
	RowClass string
	RowData  map[string]string
	RowAttr  map[string]string
	IsLess   func(*TableRow, *TableRow) bool
}

func newTableRow(data map[string]interface{}) *TableRow {
	return &TableRow{Data: data}
}

func (r *TableRow) Less(than rbtree.Item) bool {
	thanRow := than.(*TableRow)
	if r.IsLess == nil {
		panic("Missing sort funcion in TableRow")
	}

	return r.IsLess(r, thanRow)
}

func (tr *TableRow) MarshalJSON() ([]byte, error) {
	data := make(map[string]interface{}, len(tr.Data))

	if len(tr.RowId) > 0 {
		data["DT_RowId"] = tr.RowId
	}

	if len(tr.RowClass) > 0 {
		data["DT_RowClass"] = tr.RowClass
	}

	if len(tr.RowData) > 0 {
		data["DT_RowData"] = tr.RowId
	}

	if len(tr.RowAttr) > 0 {
		data["DT_RowAttr"] = tr.RowAttr
	}

	for key, val := range tr.Data {
		data[key] = val
	}
	return json.Marshal(data)
}

type SearchRequest struct {
	Draw    int                 `json:"draw"`
	Columns []*ColumnDefinition `json:"columns"`
	Start   int                 `json:"start"`
	Length  int                 `json:"length"`
	Search  *SearchDefinition   `json:"search"`
}

func (sr *SearchRequest) End() int {
	return sr.Start + sr.Length
}

func newSearchRequest(request *http.Request, params httprouter.Params) *SearchRequest {
	searchReq := &SearchRequest{}

	values, err := getRequestValues(request)
	if err != nil {
		return searchReq
	}

	formValues := TreePostFormValues(values)
	ordering := parseOrdering(formValues)

	if _, ok := formValues["columns"]; ok {
		columnsList := formValues["columns"].(map[string]interface{})
		searchReq.Columns = make([]*ColumnDefinition, len(columnsList))

		for key, column := range columnsList {
			searchReq.Columns[cast.ToInt(key)] = newColumnDefinition(column.(map[string]interface{}), DataTableOrder(ordering[key]))
		}

		searchReq.Draw = cast.ToInt(formValues["draw"])
		searchReq.Length = cast.ToInt(formValues["length"])
		searchReq.Start = cast.ToInt(formValues["start"])
		searchReq.Search = newSearchDefinition(formValues["search"].(map[string]interface{}))
	}
	return searchReq
}

func getRequestValues(request *http.Request) (url.Values, error) {

	switch request.Method {
	case "POST":
		request.ParseForm()
		return request.PostForm, nil
	case "GET":
		return request.URL.Query(), nil
	default:
		return nil, fmt.Errorf("Unsupported request method: %s", request.Method)
	}
}

func parseOrdering(formValues map[string]interface{}) map[string]DataTableOrder {
	ordering := map[string]DataTableOrder{}
	if orders, ok := formValues["order"]; ok {
		for _, val := range orders.(map[string]interface{}) {
			mapValue := val.(map[string]interface{})
			column, columnExists := mapValue["column"]
			dir, dirExists := mapValue["dir"]
			if columnExists && dirExists {
				ordering[cast.ToString(column)] = DataTableOrder(cast.ToString(dir))
			}
		}
	}

	return ordering
}

type SearchDefinition struct {
	Value         string         `json:"value"`
	Regexp        bool           `json:"regexp"`
	regExp        *regexp.Regexp `json:"-"`
	InvalidRegExp bool           `json:"-"`
}

func newSearchDefinition(search map[string]interface{}) *SearchDefinition {
	return &SearchDefinition{Value: search["value"].(string), Regexp: cast.ToBool(search["regexp"])}
}

func (sd *SearchDefinition) IsApplicable() bool {
	if sd.Regexp {
		if !sd.InvalidRegExp {
			_, err := sd.getRegExp()
			return err == nil
		}
		return false
	}
	return sd.Value != ""
}

func (sd *SearchDefinition) IsMatch(value string) bool {

	if regExp, err := sd.getRegExp(); err == nil {
		return regExp.MatchString(value)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(sd.Value))
}

func (sd *SearchDefinition) getRegExp() (*regexp.Regexp, error) {
	if !sd.Regexp {
		return nil, errors.New("Regexp not applied for search")
	}
	if sd.regExp == nil {
		if regExp, err := regexp.Compile(sd.Value); err == nil {
			sd.regExp = regExp
		} else {
			// report and turn off
			log.Printf("Cannot compile regular expression to search '%s'", sd.Value)
			sd.InvalidRegExp = true
			return nil, errors.New("Cannot compile RegExp")
		}
	}

	return sd.regExp, nil
}

type ColumnDefinition struct {
	Name       string            `json:"data"`
	Searchable bool              `json:"searchable"`
	Orderable  bool              `json:"orderable"`
	Order      DataTableOrder    `json:"-"`
	Search     *SearchDefinition `json:"search"`
}

func newColumnDefinition(column map[string]interface{}, order DataTableOrder) *ColumnDefinition {
	return &ColumnDefinition{
		Name:       column["data"].(string),
		Searchable: cast.ToBool(column["searchable"]),
		Orderable:  cast.ToBool(column["orderable"]),
		Order:      order,
		Search:     newSearchDefinition(column["search"].(map[string]interface{})),
	}
}
