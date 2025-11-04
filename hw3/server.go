package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

var DatasetPath = "dataset.xml"

type UserInfo struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	About     string `xml:"about"`
}
type ReqParams struct {
	Query      string
	OrderField string
	OrderBy    int
	Offset     int
	Limit      int
}

type UsersDB struct {
	users []User
}

func NewUsersDB(fileName string) (*UsersDB, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var users []User

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "row" {
			continue
		}
		var u UserInfo
		if err := dec.DecodeElement(&u, &se); err != nil {
			return nil, err
		}
		users = append(users, User{
			ID:     u.ID,
			Name:   strings.TrimSpace(u.FirstName + " " + u.LastName),
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		})
	}

	return &UsersDB{users: users}, nil
}

func (db *UsersDB) SearchUsers(params *ReqParams) ([]User, error) {

	// 1) проверяем query (Name, about)
	// 2) сортируем по order_field + order by (Id, Age, Name)
	// 3) применяем offset + limit

	var filtered []User
	q := strings.ToLower(strings.TrimSpace(params.Query))
	if q == "" {
		filtered = append(filtered, db.users...)
	} else {
		for _, u := range db.users {
			if strings.Contains(strings.ToLower(u.Name), q) ||
				strings.Contains(strings.ToLower(u.About), q) {
				filtered = append(filtered, u)
			}
		}
	}

	orderField := params.OrderField
	if orderField == "" {
		orderField = "Name"
	}
	switch orderField {
	case "Id", "Age", "Name":
	default:
		return nil, fmt.Errorf(ErrorBadOrderField)
	}

	if params.OrderBy == OrderByAsc || params.OrderBy == OrderByDesc {
		less := func(i, j int) bool { return false }
		switch orderField {
		case "Id":
			less = func(i, j int) bool { return filtered[i].ID < filtered[j].ID }
		case "Age":
			less = func(i, j int) bool { return filtered[i].Age < filtered[j].Age }
		case "Name":
			less = func(i, j int) bool { return filtered[i].Name < filtered[j].Name }
		}
		sort.Slice(filtered, func(i, j int) bool {
			if params.OrderBy == OrderByAsc {
				return less(i, j)
			}
			return less(j, i)
		})
	} else if params.OrderBy != OrderByAsIs {
		return nil, errors.New("bad order_by value")
	}

	if params.Offset < 0 || params.Limit < 0 {
		return nil, errors.New("negative limit/offset")
	}
	if params.Offset >= len(filtered) {
		return []User{}, nil
	}
	end := params.Offset + params.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[params.Offset:end], nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// Тут писать SearchServer
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	params, err := parseParams(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := NewUsersDB(DatasetPath)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "cannot open dataset")
		return
	}

	relevantUsers, err := db.SearchUsers(params)
	if err != nil {
		if err.Error() == ErrorBadOrderField {
			writeErrorJSON(w, http.StatusBadRequest, ErrorBadOrderField)
			return
		}
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(relevantUsers); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
}

func parseParams(r *http.Request) (*ReqParams, error) {
	params := &ReqParams{}

	params.Query = r.URL.Query().Get("query")
	params.OrderField = r.URL.Query().Get("order_field")

	order_by, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if err != nil {
		return &ReqParams{}, errors.New("cant read order_by value")
	}
	params.OrderBy = order_by

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return &ReqParams{}, errors.New("cant read limit value")
	}
	params.Limit = limit

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		return &ReqParams{}, errors.New("cant read offset value")
	}
	params.Offset = offset

	return params, nil
}

func writeErrorJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SearchErrorResponse{Error: msg})
}
