package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTempDataset(t *testing.T, xmlBody string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "dataset.xml")
	if err := os.WriteFile(p, []byte(xmlBody), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	return p
}

func startSearchServerWithDataset(t *testing.T, datasetXML string) (*httptest.Server, string) {
	t.Helper()
	path := writeTempDataset(t, datasetXML)
	DatasetPath = path
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	t.Cleanup(ts.Close)
	return ts, path
}

const smallXML = `<?xml version="1.0" encoding="UTF-8"?>
<root>
  <row>
    <id>1</id><age>30</age>
    <first_name>Alice</first_name><last_name>Zephyr</last_name>
    <gender>female</gender>
    <about>Loves Go and golang</about>
  </row>
  <row>
    <id>2</id><age>25</age>
    <first_name>Bob</first_name><last_name>Yellow</last_name>
    <gender>male</gender>
    <about>Enjoys hiking and cooking</about>
  </row>
  <row>
    <id>3</id><age>40</age>
    <first_name>Charlie</first_name><last_name>Xavier</last_name>
    <gender>male</gender>
    <about>Works on concurrency and systems</about>
  </row>
  <row>
    <id>4</id><age>28</age>
    <first_name>Donna</first_name><last_name>White</last_name>
    <gender>female</gender>
    <about>About nothing in particular</about>
  </row>
  <row>
    <id>5</id><age>35</age>
    <first_name>Evan</first_name><last_name>Violet</last_name>
    <gender>male</gender>
    <about>Golang and automation</about>
  </row>
</root>`

// Tests SearchServer

func TestSearchServer_SimpleQueryAndDefaultSort(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	// query пустой т.е. сортировка по Name
	params := url.Values{
		"query":       {""},
		"order_field": {""},
		"order_by":    {"0"},
		"limit":       {"5"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var got []User
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("want 5, got %d", len(got))
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name, got[3].Name, got[4].Name}
	want := []string{"Alice Zephyr", "Bob Yellow", "Charlie Xavier", "Donna White", "Evan Violet"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %q want %q", i, names[i], want[i])
		}
	}
}

func TestSearchServer_FilterOrderByAgeDescAndPagination(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	params := url.Values{
		"query":       {"go"},
		"order_field": {"Age"},
		"order_by":    {"-1"},
		"limit":       {"1"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var page1 []User
	if err := json.NewDecoder(resp.Body).Decode(&page1); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page1) != 1 || page1[0].Name != "Evan Violet" {
		t.Fatalf("unexpected page1: %+v", page1)
	}

	// offset=1, вторая запись должна быть Alice
	params.Set("offset", "1")
	resp2, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET2: %v", err)
	}
	defer resp2.Body.Close()
	var page2 []User
	if err := json.NewDecoder(resp2.Body).Decode(&page2); err != nil {
		t.Fatalf("decode2: %v", err)
	}
	if len(page2) != 1 || page2[0].Name != "Alice Zephyr" {
		t.Fatalf("unexpected page2: %+v", page2)
	}
}

func TestSearchServer_OrderByAsIsPreservesFileOrder(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	params := url.Values{
		"query":       {""},
		"order_field": {"Age"},
		"order_by":    {"0"},
		"limit":       {"5"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var got []User
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	ids := []int{got[0].ID, got[1].ID, got[2].ID, got[3].ID, got[4].ID}
	want := []int{1, 2, 3, 4, 5}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("as-is mismatch at %d: got %d want %d", i, ids[i], want[i])
		}
	}
}

func TestSearchServer_BadOrderField400(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)
	params := url.Values{
		"query":       {""},
		"order_field": {"Salary"},
		"order_by":    {"0"},
		"limit":       {"5"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var er SearchErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode err json: %v", err)
	}
	if er.Error != ErrorBadOrderField {
		t.Fatalf("want %q, got %q", ErrorBadOrderField, er.Error)
	}
}

func TestSearchServer_ParamParseErrors(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	// некорректный order_by
	params := url.Values{
		"limit":       {"5"},
		"offset":      {"0"},
		"order_by":    {"NaN"},
		"order_field": {"Id"},
	}
	resp, _ := http.Get(ts.URL + "?" + params.Encode())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	// некорректный limit
	params.Set("order_by", "0")
	params.Set("limit", "oops")
	resp2, _ := http.Get(ts.URL + "?" + params.Encode())
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("status2: %d", resp2.StatusCode)
	}

	// некорректный offset
	params.Set("limit", "5")
	params.Set("offset", "oops")
	resp3, _ := http.Get(ts.URL + "?" + params.Encode())
	if resp3.StatusCode != http.StatusBadRequest {
		t.Fatalf("status3: %d", resp3.StatusCode)
	}
}

// Tests FindUsers

func newClient(ts *httptest.Server) *SearchClient {
	return &SearchClient{
		AccessToken: "token",
		URL:         ts.URL,
	}
}

func TestFindUsers_HappyPathAndNextPage(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)
	cl := newClient(ts)

	// limit=1 клиент отправит 2 (limit++) и вернёт NextPage=true при 2 ответах
	resp, err := cl.FindUsers(SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "go",
		OrderField: "Age",
		OrderBy:    OrderByDesc,
	})
	if err != nil {
		t.Fatalf("FindUsers: %v", err)
	}
	if !resp.NextPage {
		t.Fatalf("want NextPage=true")
	}
	if len(resp.Users) != 1 || resp.Users[0].Name != "Evan Violet" {
		t.Fatalf("unexpected page: %+v", resp)
	}
}

func TestFindUsers_BadOrderFieldMappedToClientError(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)
	cl := newClient(ts)
	_, err := cl.FindUsers(SearchRequest{
		Limit:      3,
		Offset:     0,
		Query:      "",
		OrderField: "WrongField",
		OrderBy:    OrderByAsc,
	})
	if err == nil || !strings.Contains(err.Error(), "OrderFeld WrongField invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsers_Server500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		old := DatasetPath
		DatasetPath = "no_such_file.xml"
		defer func() { DatasetPath = old }()
		SearchServer(w, r)
	}))
	defer ts.Close()

	cl := newClient(ts)
	_, err := cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "SearchServer fatal error") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFindUsers_401Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("AccessToken") != "valid" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	cl := &SearchClient{AccessToken: "bad", URL: ts.URL}
	_, err := cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "bad AccessToken") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestFindUsers_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second + 200*time.Millisecond)
	}))
	defer ts.Close()

	cl := newClient(ts)
	_, err := cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "timeout for ") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestFindUsers_BadRequestUnknownErrorAndBadJSON(t *testing.T) {
	ts400 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(SearchErrorResponse{Error: "some error"})
	}))
	defer ts400.Close()

	cl := newClient(ts400)
	_, err := cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "unknown bad request error: some error") {
		t.Fatalf("unexpected: %v", err)
	}

	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{oops`))
	}))
	defer tsBadJSON.Close()

	cl.URL = tsBadJSON.URL
	_, err = cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "cant unpack error json: ") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestFindUsers_SuccessBadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not_array}`))
	}))
	defer ts.Close()

	cl := newClient(ts)
	_, err := cl.FindUsers(SearchRequest{Limit: 1, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "cant unpack result json: ") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestFindUsers_LimitAndOffsetValidationAndCapping(t *testing.T) {
	// limit < 0
	cl := &SearchClient{URL: "http://example.invalid"}
	_, err := cl.FindUsers(SearchRequest{Limit: -1})
	if err == nil || !strings.Contains(err.Error(), "limit must be > 0") {
		t.Fatalf("want limit error, got %v", err)
	}
	// offset < 0
	_, err = cl.FindUsers(SearchRequest{Limit: 1, Offset: -1})
	if err == nil || !strings.Contains(err.Error(), "offset must be > 0") {
		t.Fatalf("want offset error, got %v", err)
	}

	var seenLimit string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenLimit = r.URL.Query().Get("limit")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	cl.URL = ts.URL
	_, _ = cl.FindUsers(SearchRequest{Limit: 100, Offset: 0})
	if seenLimit != "26" {
		t.Fatalf("expected sent limit=26, got %s", seenLimit)
	}
}

func TestSearchServer_SortByNameAsc_And_ByIdAsc(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	params := url.Values{
		"query":       {""},
		"order_field": {"Name"},
		"order_by":    {"1"},
		"limit":       {"5"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var got []User
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	wantNames := []string{"Alice Zephyr", "Bob Yellow", "Charlie Xavier", "Donna White", "Evan Violet"}
	for i, w := range wantNames {
		if got[i].Name != w {
			t.Fatalf("Name ASC mismatch at %d: got %q want %q", i, got[i].Name, w)
		}
	}

	params.Set("order_field", "Id")
	resp2, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET2: %v", err)
	}
	defer resp2.Body.Close()
	var got2 []User
	if err := json.NewDecoder(resp2.Body).Decode(&got2); err != nil {
		t.Fatalf("decode2: %v", err)
	}
	for i, id := range []int{1, 2, 3, 4, 5} {
		if got2[i].ID != id {
			t.Fatalf("Id ASC mismatch at %d: got %d want %d", i, got2[i].ID, id)
		}
	}
}

func TestSearchServer_BadOrderByValue(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	params := url.Values{
		"query":       {""},
		"order_field": {"Id"},
		"order_by":    {"2"},
		"limit":       {"5"},
		"offset":      {"0"},
	}
	resp, err := http.Get(ts.URL + "?" + params.Encode())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var er SearchErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(er.Error, "bad order_by value") {
		t.Fatalf("unexpected error: %q", er.Error)
	}
}

func TestSearchServer_NegativeLimitAndOffset(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	// limit < 0
	params := url.Values{
		"limit":       {"-1"},
		"offset":      {"0"},
		"order_by":    {"0"},
		"order_field": {"Id"},
	}
	resp, _ := http.Get(ts.URL + "?" + params.Encode())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status limit<0: %d", resp.StatusCode)
	}

	// offset < 0
	params.Set("limit", "5")
	params.Set("offset", "-2")
	resp2, _ := http.Get(ts.URL + "?" + params.Encode())
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("status offset<0: %d", resp2.StatusCode)
	}
}

func TestSearchServer_OffsetBeyondLength_And_EndClamp(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	params := url.Values{
		"limit":       {"5"},
		"offset":      {"999"},
		"order_by":    {"0"},
		"order_field": {"Id"},
	}
	resp, _ := http.Get(ts.URL + "?" + params.Encode())
	defer resp.Body.Close()
	var got []User
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 0 {
		t.Fatalf("want empty slice, got %d", len(got))
	}

	params.Set("offset", "3")
	params.Set("limit", "10")
	resp2, _ := http.Get(ts.URL + "?" + params.Encode())
	defer resp2.Body.Close()
	var got2 []User
	_ = json.NewDecoder(resp2.Body).Decode(&got2)
	if len(got2) != 2 {
		t.Fatalf("want 2, got %d", len(got2))
	}
}

func TestSearchServer_MethodNotAllowed(t *testing.T) {
	ts, _ := startSearchServerWithDataset(t, smallXML)

	req, _ := http.NewRequest(http.MethodPost, ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

type errWriter struct {
	hdr  http.Header
	code int
}

func (e *errWriter) Header() http.Header {
	if e.hdr == nil {
		e.hdr = make(http.Header)
	}
	return e.hdr
}
func (e *errWriter) WriteHeader(status int)    { e.code = status }
func (e *errWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write fail") }

func TestSearchServer_EncodeErrorBranch(t *testing.T) {
	_, _ = startSearchServerWithDataset(t, smallXML)
	req := httptest.NewRequest(http.MethodGet, "/?limit=5&offset=0&order_by=0&order_field=Id", nil)
	w := &errWriter{}
	SearchServer(w, req)
	if w.code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.code)
	}
}

func TestNewUsersDB_MalformedXML(t *testing.T) {
	path := writeTempDataset(t, `<root><row><id>1</id><first_name>Bad`)
	_, err := NewUsersDB(path)
	if err == nil {
		t.Fatalf("expected XML decode error")
	}
}
