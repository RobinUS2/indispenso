package data_table

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/url"
)

func DataStoreHandler(tableStore TableStore) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		request := newSearchRequest(r, ps)
		result := tableStore.QueryData(request)
		jsonBytes, _ := json.Marshal(result)
		fmt.Fprint(w, string(jsonBytes))
	}
}

func TreePostFormValues(values url.Values) map[string]interface{} {
	res := make(map[string]interface{})
	var currValue map[string]interface{}
	for rawKey, value := range values {
		if vs := value; len(vs) > 0 {
			currValue = res
			keyPath := ParseKey(rawKey)
			lastIndex := len(keyPath) - 1
			for index, key := range keyPath {
				if index == lastIndex {
					currValue[key] = vs[0]
				} else {
					if _, ok := currValue[key]; !ok {
						currValue[key] = make(map[string]interface{})
					}
					currValue = currValue[key].(map[string]interface{})
				}
			}
		}

	}
	return res
}

func ParseKey(key string) []string {
	res := make([]string, 0)
	var currKey bytes.Buffer

	for _, char := range key {
		if char == '[' || char == ']' {
			if currKey.Len() > 0 {
				res = append(res, currKey.String())
				currKey.Reset()
			}
		} else {
			currKey.WriteRune(char)
		}
	}

	if currKey.Len() > 0 {
		res = append(res, currKey.String())
	}

	return res
}
