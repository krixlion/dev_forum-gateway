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

var _ http.Handler = (*AuthHandler)(nil)

// AuthHandler handles all `/auth` endpoints.
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

// registerRoutes registers handlers and middleware for each route.
// All routes are registered on '/' so that the handler can be mounted on any path.
func (s AuthHandler) registerRoutes() {
	s.router.With(otelhttp.NewMiddleware("SignIn")).Post("/sign-in", httpe.ToHandlerFunc(s.SignIn, s.logger))
	s.router.With(otelhttp.NewMiddleware("SignOut")).Post("/sign-out", httpe.ToHandlerFunc(s.SignOut, s.logger))
	s.router.With(otelhttp.NewMiddleware("GetAccessToken")).Post("/get-access-token", httpe.ToHandlerFunc(s.GetAccessToken, s.logger))
}

// ServeHTTP is called on each request before it's passed to the handler.
func (s AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// SignInRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type SignInRequest struct {
	Email    string `json:"email,omitempty" example:"example@gmail.com"`
	Password string `json:"password,omitempty" example:"zaq1@WSXEDC"`
}

// SignInResponse exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type SignInResponse struct {
	RefreshToken string `json:"refresh_token,omitempty" example:"dfr_YWRpQWNrbURTZHZmQ1lhZF9jOWIyZDA2Mg=="`
}

// SignIn retrieves a new refresh token from the AuthService for a user with given credentials.
//
//	@Id			SignIn
//	@Tags		auth
//	@Summary	"Sign in"
//	@Router		/auth/sign-in	[post]
//	@Param		payload			body	SignInRequest	true	"Payload"
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	SignInResponse
//	@Failure	400	"Payload could not be parsed or credentials are missing/empty."
//	@Failure	401	"Credentials are invalid."
//	@Failure	500	"An unexpected error occurred."
func (s AuthHandler) SignIn(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	req := SignInRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	if req.Password == "" {
		return nil, httpe.NewError(http.StatusBadRequest, "password is missing or empty")
	}

	if req.Email == "" {
		return nil, httpe.NewError(http.StatusBadRequest, "email is missing or empty")
	}

	resp, err := s.authService.SignIn(ctx, &pb.SignInRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.FailedPrecondition {
			return nil, httpe.NewError(http.StatusUnauthorized, "invalid email or password")
		}
		s.logger.Log(ctx, "Failed to sign in", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, SignInResponse{RefreshToken: resp.GetRefreshToken()}), nil
}

// SignOutRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type SignOutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty" example:"dfr_YWRpQWNrbURTZHZmQ1lhZF9jOWIyZDA2Mg=="`
}

// SignOut signs out a user assigned to a given refresh token.
//
//	@Id			SignOut
//	@Tags		auth
//	@Summary	"Sign out"
//	@Router		/auth/sign-out	[post]
//	@Param		payload			body	SignOutRequest	true	"Payload"
//	@Accept		json
//	@Success	200
//	@Failure	400	"Payload could not be parsed or the refresh token is missing/empty."
//	@Failure	500	"An unexpected error occurred."
func (s AuthHandler) SignOut(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	req := SignOutRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	if req.RefreshToken == "" {
		return nil, httpe.NewError(http.StatusBadRequest, "refresh_token is missing or empty")
	}

	if _, err := s.authService.SignOut(ctx, &pb.SignOutRequest{RefreshToken: req.RefreshToken}); err != nil {
		s.logger.Log(ctx, "Failed to sign out", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, nil), nil
}

// GetAccessTokenRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type GetAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token,omitempty" example:"dfr_YWRpQWNrbURTZHZmQ1lhZF9jOWIyZDA2Mg=="`
}

// GetAccessTokenResponse exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type GetAccessTokenResponse struct {
	AccessToken string `json:"access_token,omitempty" example:"dfa_Zk5QTldQc0Vtb2ZOUFduTl9iZjNmMTJlYw=="`
}

// GetAccessToken retrieves a new access token from the AuthService based on a given refresh token.
//
//	@Id			GetAccessToken
//	@Tags		auth
//	@Summary	"Get a new access token."
//	@Router		/auth/get-access-token	[post]
//	@Param		payload					body	GetAccessTokenRequest	true	"Payload"
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	GetAccessTokenResponse
//	@Failure	400	"Payload could not be parsed or the refresh token is missing/empty."
//	@Failure	500	"An unexpected error occurred."
func (s AuthHandler) GetAccessToken(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	req := GetAccessTokenRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to parse request body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	if req.RefreshToken == "" {
		return nil, httpe.NewError(http.StatusBadRequest, "refresh_token is missing or empty")
	}

	resp, err := s.authService.GetAccessToken(ctx, &pb.GetAccessTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		s.logger.Log(ctx, "Failed to get access token", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, GetAccessTokenResponse{AccessToken: resp.GetAccessToken()}), nil
}
