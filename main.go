package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/belimawr/url-shortener/db"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type config struct {
	Port      int    `env:"HTTP_PORT" envDefault:"3000"`
	DBConnStr string `env:"DB_CONN_STR" envDefault:"host=localhost user=db_user dbname=url sslmode=disable"`
}

type database interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
}

type handler struct {
	db database
}

func NewHandler(db database) handler {
	return handler{db: db}
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

	logger = logger.Output(zerolog.NewConsoleWriter())

	dbConn, err := sql.Open("postgres", cfg.DBConnStr)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to databse")
	}

	if err := dbConn.Ping(); err != nil {
		logger.Fatal().Err(err).Msg("cannot ping database")
	}

	h := NewHandler(db.NewPostgres(dbConn))
	//	h := NewHandler(inMemory{})

	r := chi.NewRouter()
	r.Use(hlog.NewHandler(logger))

	r.Use(RequestIDHandler("req_id", "Request-Id"))
	r.Use(hlog.RemoteAddrHandler("ip"))
	r.Use(hlog.UserAgentHandler("user_agent"))
	r.Use(hlog.RefererHandler("referer"))
	r.Use(requestLogger)

	r.Get("/save", h.saveURL)
	r.Get("/go", h.getURL)
	r.Get("/echo", echo)
	r.Get("/rec", recusion)

	logger.Info().Msgf("starting HTTP server at port %d", cfg.Port)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r))
}

func (h handler) saveURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)
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

	// Save URL in our database
	if err := h.db.Set(ctx, key, parsedURL.String()); err != nil {
		logger.Error().Err(err).Msg("cannot insert into postgres database")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	msg := fmt.Sprintf("Your new url is: %s", key)
	w.Write([]byte(msg))
	logger.Info().Msgf("saveURL %s %s", key, parsedURL)
}

func (h handler) getURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)
	// /go?to=uuidv4
	key := r.URL.Query().Get("to")

	// SQL DB
	strURL, err := h.db.Get(ctx, key)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug().Msgf("cannot find URL for token: '%s'", key)
			http.NotFound(w, r)
			return
		}

		logger.Error().Err(err).Msg("cannot read data from database")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	http.Redirect(w, r, strURL, http.StatusTemporaryRedirect)
	logger.Info().Msgf("getURL %s %s", key, strURL)
}

func requestLogger(next http.Handler) http.Handler {
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
