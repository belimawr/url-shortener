package middleware

import (
	"context"
	"net/http"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())

		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			c = c.Str("url_path", r.URL.Path)
			c = c.Str("http_method", r.Method)
			c = c.Str("ip", r.RemoteAddr)
			c = c.Str("user_agent", r.UserAgent())
			return c
		})

		next.ServeHTTP(w, r)

		logger.Info().Msgf("%s %s", r.Method, r.URL.String())
	}

	return http.HandlerFunc(fn)
}

func requestID(header string, r *http.Request) string {
	id := r.Header.Get(header)

	if id == "" {
		id = xid.New().String()
	}

	return id
}

func RequestID(fieldKey, headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := zerolog.Ctx(ctx)

			id := requestID(headerName, r)

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
