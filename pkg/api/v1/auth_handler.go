package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	pb "github.com/krixlion/dev_forum-auth/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	router      chi.Router
	authService pb.AuthServiceClient
	logger      logging.Logger
}

func MakeAuthHandler(grpcClient pb.AuthServiceClient, logger logging.Logger) AuthHandler {
	s := AuthHandler{
		router:      chi.NewRouter(),
		authService: grpcClient,
		logger:      logger,
	}
	s.registerRoutes()
	return s
}

func (s AuthHandler) registerRoutes() {
	s.router.With(otelhttp.NewMiddleware("SignIn")).Post("/sign-in", httpe.ToHandlerFunc(s.SignIn, s.logger))
	s.router.With(otelhttp.NewMiddleware("SignOut")).Post("/sign-out", httpe.ToHandlerFunc(s.SignOut, s.logger))
	s.router.With(otelhttp.NewMiddleware("GetAccessToken")).Post("/get-access-token", httpe.ToHandlerFunc(s.GetAccessToken, s.logger))
}

func (s AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s AuthHandler) SignIn(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	body := make(map[string]string)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	password, ok := body["password"]
	if !ok {
		return nil, httpe.NewError(http.StatusBadRequest, "missing password")
	}

	email, ok := body["email"]
	if !ok {
		return nil, httpe.NewError(http.StatusBadRequest, "missing email")
	}

	resp, err := s.authService.SignIn(ctx, &pb.SignInRequest{Email: email, Password: password})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.FailedPrecondition {
			return nil, httpe.NewError(http.StatusUnauthorized, "invalid email or password")
		}
		s.logger.Log(ctx, "Failed to sign in", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, map[string]string{"refresh_token": resp.GetRefreshToken()}), nil
}

func (s AuthHandler) SignOut(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	body := make(map[string]string)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	refreshToken, ok := body["refresh_token"]
	if !ok {
		return nil, httpe.NewError(http.StatusBadRequest, "missing refresh_token")
	}

	if _, err := s.authService.SignOut(ctx, &pb.SignOutRequest{RefreshToken: refreshToken}); err != nil {
		s.logger.Log(ctx, "Failed to sign out", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, nil), nil
}

func (s AuthHandler) GetAccessToken(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	body := make(map[string]string)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	refreshToken, ok := body["refresh_token"]
	if !ok {
		return nil, httpe.NewError(http.StatusBadRequest, "missing refresh_token")
	}

	resp, err := s.authService.GetAccessToken(ctx, &pb.GetAccessTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		s.logger.Log(ctx, "Failed to get access token", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, map[string]string{"access_token": resp.GetAccessToken()}), nil
}
