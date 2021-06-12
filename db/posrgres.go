package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
)

func NewPostgres(conn *sql.DB) Postgres {
	return Postgres{
		db: conn,
	}
}

type Postgres struct {
	db *sql.DB
}

func (p Postgres) Get(ctx context.Context, key string) (string, error) {
	logger := zerolog.Ctx(ctx)
	var requestedURL string
	row := p.db.QueryRowContext(ctx, "SELECT url FROM urls WHERE token=$1", key)

	if err := row.Scan(&requestedURL); err != nil {
		if err == sql.ErrNoRows {
			logger.Debug().Msgf("cannot find URL for token: '%s'", key)
			return "", fmt.Errorf("no entry for key %q", key)
		}

		logger.Error().Err(err).Msg("cannot read data from database")
		return "", fmt.Errorf("cannot read data from database: %w", err)
	}

	return requestedURL, nil
}

func (p Postgres) Set(ctx context.Context, key, value string) error {
	logger := zerolog.Ctx(ctx)

	_, err := p.db.ExecContext(ctx, "INSERT INTO urls (token, url) VALUES ($1, $2);", key, value)
	if err != nil {
		logger.Error().Err(err).Msg("cannot insert into sql database")
		return fmt.Errorf("cannot insert data: %w", err)
	}

	return nil
}
