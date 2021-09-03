package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Dataset struct {
	XMLName xml.Name `xml:"root"`
	Text    string   `xml:",chardata"`
	Row     []struct {
		Text          string `xml:",chardata"`
		ID            int    `xml:"id"`
		Guid          string `xml:"guid"`
		IsActive      bool   `xml:"isActive"`
		Balance       string `xml:"balance"`
		Picture       string `xml:"picture"`
		Age           int    `xml:"age"`
		EyeColor      string `xml:"eyeColor"`
		FirstName     string `xml:"first_name"`
		LastName      string `xml:"last_name"`
		Gender        string `xml:"gender"`
		Company       string `xml:"company"`
		Email         string `xml:"email"`
		Phone         string `xml:"phone"`
		Address       string `xml:"address"`
		About         string `xml:"about"`
		Registered    string `xml:"registered"`
		FavoriteFruit string `xml:"favoriteFruit"`
	} `xml:"row"`
}

type TestCase struct {
	Request      SearchRequest
	IsError      bool
	ErrorMessage string
}

var dataset Dataset

func TestMain(m *testing.M) {
	file, err := os.Open("dataset.xml")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	data, err := ioutil.ReadAll(file)

	if err != nil {
		log.Fatal(err)
	}

	err = xml.Unmarshal(data, &dataset)

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		msg := fmt.Sprintf("Error when parsing limit: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		msg := fmt.Sprintf("Error when parsing offset: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(io.Discard, "offset: %v\n", offset)

	query := r.FormValue("query")
	orderField := r.FormValue("order_field")

	fmt.Fprintf(io.Discard, "orderField: %v\n", orderField)

	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		msg := fmt.Sprintf("Error when parsing order_by: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(io.Discard, "orderBy: %v\n", orderBy)

	var result []User
	for _, row := range dataset.Row {
		name := row.FirstName + " " + row.LastName
		if strings.Contains(fmt.Sprint(row), query) {
			result = append(result, User{
				Id:     row.ID,
				Name:   name,
				Age:    row.Age,
				About:  row.About,
				Gender: row.Gender,
			})
		}
	}

	switch orderField {
	case "Id":
		if orderBy == OrderByAsc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Id < result[q].Id
			})
		}

		if orderBy == OrderByDesc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Id > result[q].Id
			})
		}
	case "Name", "":
		if orderBy == OrderByAsc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Name < result[q].Name
			})
		}

		if orderBy == OrderByDesc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Name > result[q].Name
			})
		}

	case "Age":
		if orderBy == OrderByAsc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Age < result[q].Age
			})
		}

		if orderBy == OrderByDesc {
			sort.SliceStable(result, func(p, q int) bool {
				return result[p].Age > result[q].Age
			})
		}
	default:
		message, _ := json.Marshal(SearchErrorResponse{Error: "ErrorBadOrderField"})
		http.Error(w, string(message), http.StatusBadRequest)
		return
	}

	if offset > len(result) {
		offset = len(result)
	}

	if limit == 0 {
		limit = len(result)
	}

	if offset+limit > len(result) {
		limit = len(result) - offset
	}

	result = result[offset : limit+offset]

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(result)

	if err != nil {
		panic(err)
	}

}

func TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	testCases := []TestCase{
		{
			Request: SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			IsError:      false,
			ErrorMessage: "",
		},
		{
			Request: SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			IsError:      true,
			ErrorMessage: "Limit must be >= than 0",
		},
		{
			Request: SearchRequest{
				Limit:      2,
				Offset:     -1,
				Query:      "Hilda",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			IsError:      true,
			ErrorMessage: "Offset must be >= 0",
		},
		{
			Request: SearchRequest{
				Limit:      26,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			IsError:      false,
			ErrorMessage: "",
		},
		{
			Request: SearchRequest{
				Limit:      0,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			IsError:      false,
			ErrorMessage: "",
		},
		{
			Request: SearchRequest{
				Limit:      1000,
				Offset:     0,
				Query:      "",
				OrderField: "jjjj",
				OrderBy:    0,
			},
			IsError:      true,
			ErrorMessage: "bad orderfield",
		},
	}

	client := &SearchClient{
		AccessToken: "lol",
		URL:         ts.URL,
	}

	for _, tc := range testCases {
		_, err := client.FindUsers(tc.Request)

		if err != nil && !tc.IsError {
			t.Error(err, tc.ErrorMessage)
		}
	}

}

func TestFindUsersStatusUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      12,
			Offset:     0,
			Query:      "Hilda",
			OrderField: "Id",
			OrderBy:    OrderByAsc,
		},
		IsError:      true,
		ErrorMessage: "no token",
	}

	client := &SearchClient{
		AccessToken: "",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf(tc.ErrorMessage)
	}
}

func TestFindUsersCannotUnpackJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("NotJSON"))
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      1111,
			Offset:     1111,
			Query:      "11111",
			OrderField: "1111",
			OrderBy:    111111,
		},
		IsError:      true,
		ErrorMessage: "not json",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf("Unexpected success with bad JSON")
	}
}

func TestFindUsersTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(""))
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      1111,
			Offset:     1111,
			Query:      "11111",
			OrderField: "1111",
			OrderBy:    111111,
		},
		IsError:      true,
		ErrorMessage: "not json",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf("Unexpected success with bad JSON")
	}
}

func TestFindUsersUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(""))
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      1111,
			Offset:     1111,
			Query:      "11111",
			OrderField: "1111",
			OrderBy:    111111,
		},
		IsError:      true,
		ErrorMessage: "bad url",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         "ftp://192.168.1.1",
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf(tc.ErrorMessage)
	}
}

func TestFindUsersStatusBadResuestErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "badJSON", http.StatusBadRequest)
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      12,
			Offset:     0,
			Query:      "dddd",
			OrderField: "dfg",
			OrderBy:    OrderByDesc,
		},
		IsError:      true,
		ErrorMessage: "bad json in error while ordering",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf(tc.ErrorMessage)
	}
}

func TestFindUsersStatusBadResuestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		message, _ := json.Marshal(SearchErrorResponse{Error: "LOLKEK"})
		http.Error(w, string(message), http.StatusBadRequest)
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      12,
			Offset:     0,
			Query:      "dddd",
			OrderField: "dfg",
			OrderBy:    OrderByDesc,
		},
		IsError:      true,
		ErrorMessage: "unknown error while ordering",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf(tc.ErrorMessage)
	}
}

func TestFindUsersInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bachok potik", http.StatusInternalServerError)
	}))
	defer ts.Close()

	tc := TestCase{
		Request: SearchRequest{
			Limit:      1111,
			Offset:     1111,
			Query:      "11111",
			OrderField: "1111",
			OrderBy:    111111,
		},
		IsError:      true,
		ErrorMessage: "not json",
	}

	client := &SearchClient{
		AccessToken: "kek",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf(tc.ErrorMessage)
	}
}
