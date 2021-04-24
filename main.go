package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
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

	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("foo", "bar").
		Logger()

	h := handler{db: db{}}

	r := chi.NewRouter()
	r.Use(hlog.NewHandler(logger))

	r.Use(RequestIDHandler("req_id", "Request-Id"))
	r.Use(hlog.RemoteAddrHandler("ip"))
	// r.Use(hlog.UserAgentHandler("user_agent"))
	// r.Use(hlog.RefererHandler("referer"))

	r.Use(middleware)

	r.Get("/save", h.saveURL)
	r.Get("/go", h.getURL)
	r.Get("/echo", echo)
	r.Get("/rec", recusion)

	logger.Info().Msgf("starting HTTP server at port %d", cfg.Port)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r))
}

func (h handler) saveURL(w http.ResponseWriter, r *http.Request) {
	logger := zerolog.Ctx(r.Context())
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
	logger.Info().Msgf("saveURL %s %s", key, parsedURL)
}

func (h handler) getURL(w http.ResponseWriter, r *http.Request) {
	logger := zerolog.Ctx(r.Context())
	// /go?to=uuidv4
	key := r.URL.Query().Get("to")
	requestedURL := h.db.Get(key)
	if requestedURL == nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, requestedURL.String(), http.StatusTemporaryRedirect)
	logger.Info().Msgf("getURL %s %s", key, requestedURL.String())
}

func middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		logger.Info().Msgf("Request started, URL: %s, mehtod: %s", r.URL.String(), r.Method)
		next.ServeHTTP(w, r)
		logger.Info().Msg("Request finished")
	}
	return http.HandlerFunc(fn)
}

func RequestID(header string, r *http.Request) string {
	id := r.Header.Get(header)

	if id == "" {
		id = xid.New().String()
	}

	return id
}

func RequestIDHandler(fieldKey, headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := zerolog.Ctx(ctx)

			id := RequestID(headerName, r)

			ctx = context.WithValue(ctx, "requestID", id)
			r = r.WithContext(ctx)

			logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str(fieldKey, id)
			})

			w.Header().Set(headerName, id)

			next.ServeHTTP(w, r)
		})
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	data, err := httputil.DumpRequest(r, true)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("cannot dump request")
	}

	w.Write(data)
}

func recusion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:3000/echo", nil)
	if err != nil {
		logger.Error().Err(err).Msg("cannot create request")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	req.Header.Set("Request-Id", RequestID("Request-Id", r))
	req.Header.Set("foo", "bar")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("could not execute request out to:", req.URL.String())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msgf("could not read response body")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	w.Write(data)
}
