package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type Repo interface {
	LoginUser(name, pass string) (jwt.Token, error)
}

type UserDB struct {
	mp map[string]string
	mu sync.Mutex
}

func NewDB() Repo {
	return &UserDB{
		mp : make(map[string]string),
		mu : sync.Mutex{},
	}
}

func(udb *UserDB) LoginUser(name, pass string) (jwt.Token, error) {
	udb.mu.Lock()
	defer udb.mu.Unlock()

	if _, ok := udb.mp[name]; !ok {
		udb.mp[name] = pass
	}

	// jwt create
	jwt 

	return 
}

func createJWT() {

}

type User struct {
	Name string `json:"username"`
	Pass string `json:"password"`
}

type UserHandl struct {
	repo Repo
}

func (uh *UserHandl) loginHandler(w http.ResponseWriter, r *http.Request) {
	usrInfo := &User{}
	json.NewDecoder(r.Body).Decode(usrInfo)

	jwtToken, err := uh.repo.LoginUser(usrInfo.Name, usrInfo.Pass)
	if err != nil {
		// todo
	}



}

func main() {
	router := mux.NewRouter()

	staticPath := filepath.Join("redditclone", "static")
	htmlPath := filepath.Join("redditclone", "static", "html")

	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(staticPath)),
		),
	)

	router.PathPrefix("POST api/login/").HandlerFunc(loginHandler)

	router.PathPrefix("/").Handler(http.FileServer(http.Dir(htmlPath)))

	server := http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("starting server at :8080")

	server.ListenAndServe()
}
