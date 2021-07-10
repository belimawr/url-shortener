package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/belimawr/url-shortener/db"
	"github.com/belimawr/url-shortener/handlers"
	"github.com/belimawr/url-shortener/middleware"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type config struct {
	Port      int    `env:"HTTP_PORT" envDefault:"3000"`
	DBConnStr string `env:"DB_CONN_STR" envDefault:"host=localhost user=db_user dbname=url sslmode=disable"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(fmt.Sprintf("cannot parse configuration: %s", err.Error()))
	}

	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Logger()

	logger = logger.Output(zerolog.NewConsoleWriter())

	dbConn, err := sql.Open("postgres", cfg.DBConnStr)
	if err != nil {
		logger.Panic().Err(err).Msg("cannot connect to databse")
	}

	if err := dbConn.Ping(); err != nil {
		logger.Panic().Err(err).Msg("cannot ping database")
	}

	//  h := NewHandler(db.NewPostgres(dbConn))
	h := handlers.New(db.NewPostgres(dbConn))

	r := chi.NewRouter()
	r.Use(hlog.NewHandler(logger))

	r.Use(middleware.RequestID("req_id", "Request-Id"))
	r.Use(middleware.RequestLogger)

	r.Get("/", handlers.Hello)
	r.Get("/save", h.SaveURL)
	r.Get("/go", h.GetURL)

	logger.Info().Msgf("starting HTTP server at port %d", cfg.Port)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r))
}
