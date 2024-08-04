package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
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
func Auth(translator tokens.Translator) func(handlerFunc httpe.HandlerEFunc) httpe.HandlerEFunc {
	return func(handlerFunc httpe.HandlerEFunc) httpe.HandlerEFunc {
		return httpe.HandlerEFunc(func(r *http.Request) (httpe.Response, error) {
			ctx := r.Context()
			bearer, ok := r.Header["Authorization"]
			if !ok {
				return nil, httpe.NewError(http.StatusUnauthorized, "Authorization header is missing")
			}

			if len(bearer) <= 0 || bearer[0] == "" {
				return nil, httpe.NewError(http.StatusUnauthorized, "Bearer token is missing")
			}

			opaqueToken, found := strings.CutPrefix(bearer[0], "Bearer ")
			if !found {
				return nil, httpe.NewError(http.StatusUnauthorized, "Bearer token is malformed")
			}

			token, err := translator.TranslateAccessToken(ctx, opaqueToken)
			if err != nil {
				return nil, httpe.NewError(http.StatusUnauthorized, "Bearer token is invalid")
			}

			if token == "" {
				return nil, httpe.NewError(http.StatusInternalServerError, "Failed to parse bearer token")
			}

			return handlerFunc(r.WithContext(context.WithValue(ctx, CtxTokenKey{}, token)))
		})
	}
}
