package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/caarlos0/env/v6"
	"github.com/google/uuid"
)

type config struct {
	Port int `env:"HTTP_PORT" envDefault:"3000"`
}

type db map[string]*url.URL

func (d db) Set(key string, value *url.URL) {
	d[key] = value
}

func (d db) Get(key string) *url.URL {
	return d[key]
}

type handler struct {
	db db
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(fmt.Sprintf("cannot parse configuration: %s", err.Error()))
	}

	h := handler{db: db{}}

	http.HandleFunc("/save", h.saveURL)
	http.HandleFunc("/go", h.getURL)

	log.Printf("starting HTTP server at port %d", cfg.Port)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil))
}

func (h handler) saveURL(w http.ResponseWriter, r *http.Request) {
	// /save?url=http://golang.org
	v := r.URL.Query().Get("url")
	key := uuid.NewString()

	parsedURL, err := url.Parse(v)
	if err != nil {
		log.Printf("ERR: %s", err.Error())

		msg := fmt.Sprintf("Could not parse URL: %q", v)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))

		return
	}

	h.db[key] = parsedURL

	msg := fmt.Sprintf("Your new url is: %s", key)
	w.Write([]byte(msg))
	log.Println("saveURL", key, parsedURL)
}

func (h handler) getURL(w http.ResponseWriter, r *http.Request) {
	// /go?to=uuidv4
	key := r.URL.Query().Get("to")
	requestedURL := h.db.Get(key)
	if requestedURL == nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, requestedURL.String(), http.StatusTemporaryRedirect)
	log.Println("getURL", key, requestedURL.String())
}
