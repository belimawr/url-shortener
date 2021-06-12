package db

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

func NewInMemory() InMemory {
	return InMemory{}
}

type InMemory map[string]string

func (d InMemory) Set(ctx context.Context, key, value string) error {
	logger := zerolog.Ctx(ctx)

	_, exists := d[key]
	if exists {
		logger.Error().Msgf("key %q already exists", key)
		return fmt.Errorf("key %q already exists", key)
	}

	d[key] = value

	return nil
}

func (d InMemory) Get(ctx context.Context, key string) (string, error) {
	logger := zerolog.Ctx(ctx)
	value, exists := d[key]
	if !exists {
		logger.Error().Msgf("key %q does not exist", key)
		return "", fmt.Errorf("key %q does not exist", key)
	}

	return value, nil
}
