package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	pb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	router      chi.Router
	userService pb.UserServiceClient
	logger      logging.Logger
	tracer      trace.Tracer
}

func MakeUserHandler(grpcClient pb.UserServiceClient, tracer trace.Tracer, logger logging.Logger) UserHandler {
	s := UserHandler{
		router:      chi.NewRouter(),
		userService: grpcClient,
		logger:      logger,
		tracer:      tracer,
	}
	s.registerRoutes()
	return s
}

func (s UserHandler) registerRoutes() {
	s.router.Method(http.MethodGet, "/{id}", otelhttp.NewHandler(httpe.NewHandlerFunc(s.GetUser, s.logger), "GetUser"))
	s.router.Method(http.MethodGet, "/", otelhttp.NewHandler(httpe.NewHandlerFunc(s.GetUsers, s.logger), "GetUsers"))
	s.router.Method(http.MethodPost, "/", otelhttp.NewHandler(httpe.NewHandlerFunc(s.CreateUser, s.logger), "CreateUser"))
	s.router.Method(http.MethodPatch, "/{id}", otelhttp.NewHandler(httpe.NewHandlerFunc(s.UpdateUser, s.logger), "UpdateUser"))
	s.router.Method(http.MethodDelete, "/{id}", otelhttp.NewHandler(httpe.NewHandlerFunc(s.DeleteUser, s.logger), "DeleteUser"))
}

// ServeHTTP is called on each request before it's passed to the handler.
func (s UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetUser retrieves public info of a user with given ID from the UserService.
//   - Returns 200 if no error is encountered.
//   - Returns 404 if the ID is an empty string or the service returned code NotFound.
//   - Returns 500 on any error.
func (s UserHandler) GetUser(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	userId := r.PathValue("id")
	if errMsg := "user not found"; userId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	resp, err := s.userService.Get(ctx, &pb.GetUserRequest{Id: userId})
	if err != nil {
		tracing.SetSpanErr(span, err)
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return nil, httpe.NewError(http.StatusNotFound, "user not found")
		}
		s.logger.Log(ctx, "Failed to get user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, resp.GetUser()), nil
}

// GetUsers queries users public info mathing given filter from the UserService.
// Reads pagination offset, limit and query filter from URL query params and
// forwards them to the UserService.
//   - Returns 200 if no error is encountered.
//   - Returns 500 on any error.
func (s UserHandler) GetUsers(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	queryParams := r.URL.Query()
	offset := queryParams.Get("offset")
	limit := queryParams.Get("limit")
	filter := queryParams.Get("filter")

	stream, err := s.userService.GetStream(ctx, &pb.GetUsersRequest{Offset: offset, Limit: limit, Filter: filter})
	if err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to init user stream", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	var users []*pb.User
	for {
		user, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			tracing.SetSpanErr(span, err)
			s.logger.Log(ctx, "Failed to read user from stream", "transport", "grpc", "err", err)
			return nil, httpe.NewGenericError(http.StatusInternalServerError)
		}
		users = append(users, user)
	}

	return httpe.NewResponse(http.StatusOK, users), nil
}

// CreateUser creates a user in the UserService.
//   - Returns 201 if no error is encountered.
//   - Returns 400 when an error is encountered when decoding request's body.
//   - Returns 500 on any other error.
func (s UserHandler) CreateUser(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	user := &pb.User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	resp, err := s.userService.Create(ctx, &pb.CreateUserRequest{User: user})
	if err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to create user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusCreated, map[string]string{"id": resp.Id}), nil
}

// UpdateUser updates an existing user in the UserService.
//   - Returns 200 if no error is encountered.
//   - Returns 404 if the ID is an empty string.
//   - Returns 400 when an error is encountered when decoding request's body.
//   - Returns 500 on any other error.
func (s UserHandler) UpdateUser(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	userId := r.PathValue("id")
	if errMsg := "user not found"; userId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	user := &pb.User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	user.Id = userId

	if _, err := s.userService.Update(ctx, &pb.UpdateUserRequest{User: user}); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to update user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusOK, nil), nil
}

// DeleteUser deletes an existing user in the UserService.
//   - Returns 204 if no error is encountered.
//   - Returns 404 if the ID is an empty string.
//   - Returns 500 on any other error.
func (s UserHandler) DeleteUser(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	userId := r.PathValue("id")
	if errMsg := "user not found"; userId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	if _, err := s.userService.Delete(ctx, &pb.DeleteUserRequest{Id: userId}); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to delete user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusNoContent, nil), nil
}
