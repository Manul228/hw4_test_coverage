package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return
}

func TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	request := SearchRequest{
		Limit:      12,
		Offset:     0,
		Query:      "Dillard",
		OrderField: "Id",
		OrderBy:    OrderByAsc,
	}
	client := &SearchClient{
		AccessToken: "lol",
		URL:         ts.URL,
	}

	result, err := client.FindUsers(request)

	t.Error(err, result)
}
