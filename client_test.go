package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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

	query := r.FormValue("query")
	var result []User
	for _, row := range dataset.Row {
		name := row.FirstName + " " + row.LastName
		if strings.Contains(name, query) {
			result = append(result, User{
				Id:     row.ID,
				Name:   name,
				Age:    row.Age,
				About:  row.About,
				Gender: row.Gender,
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(result)

	if err != nil {
		panic(err)
	}

	// http.Error(w, "Unauthorized", http.StatusUnauthorized)
	// return
}

func TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	testCases := []TestCase{
		TestCase{
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
	}

	client := &SearchClient{
		AccessToken: "lol",
		URL:         ts.URL,
	}

	for _, tc := range testCases {
		result, err := client.FindUsers(tc.Request)

		if err != nil {
			t.Error(err, result)
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
		ErrorMessage: "",
	}

	client := &SearchClient{
		AccessToken: "",
		URL:         ts.URL,
	}

	_, err := client.FindUsers(tc.Request)

	if err == nil {
		t.Errorf("Unexpected success with auth")
	}
}
