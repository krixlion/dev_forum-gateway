package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-lib/logging"
	"google.golang.org/grpc/metadata"
)

// CtxTokenKey is used to extract translated token from request context.
// Created only to avoid collisions with other packages when using context.WithValue().
// Example:
//
//	token, ok := ctx.Value(CtxTokenKey{}).(string)
//	if !ok {
//		return nil, errors.New("unexpected type")
//	}
type CtxTokenKey struct{}

// ConvertTokenContext reads the token from given context and appends it to gRPC metadata.
// Returns outgoing context with bearer token injected into Authorization header or a non-nil error.
func ConvertTokenContext(ctx context.Context) (context.Context, error) {
	token, ok := ctx.Value(CtxTokenKey{}).(string)
	if !ok {
		return nil, errors.New("token is not a string")
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("Authorization", "Bearer "+token)), nil
}

// Auth returns Middleware which validates the Bearer token extracted from
// the Authorization header. If the token is missing or invalid, it responds with 401.
// If the token is valid, the translated token is added to the request's context
// using context.WithValue().
// Use r.Context().Value(middleware.TokenKey) to extract the token.
func Auth(translator tokens.Translator, logger logging.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			bearer, ok := r.Header["Authorization"]
			if !ok {
				respond(ctx, w, http.StatusUnauthorized, "Authorization header is missing", logger)
				return
			}

			if len(bearer) <= 0 || bearer[0] == "" {
				respond(ctx, w, http.StatusUnauthorized, "Bearer token is missing", logger)
				return
			}

			opaqueToken, found := strings.CutPrefix(bearer[0], "Bearer ")
			if !found {
				respond(ctx, w, http.StatusUnauthorized, "Bearer token is malformed", logger)
				return
			}

			token, err := translator.TranslateAccessToken(ctx, opaqueToken)
			if err != nil {
				respond(ctx, w, http.StatusUnauthorized, "Bearer token is invalid", logger)
				return
			}

			if token == "" {
				respond(ctx, w, http.StatusInternalServerError, "Failed to parse bearer token", logger)
				return
			}

			h.ServeHTTP(w, r.WithContext(context.WithValue(ctx, CtxTokenKey{}, token)))
		})
	}
}

func respond(ctx context.Context, w http.ResponseWriter, status int, body string, logger logging.Logger) {
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		logger.Log(ctx, "failed to write response: %v", err)
	}
}
