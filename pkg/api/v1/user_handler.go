package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe/middleware"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	pb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ http.Handler = (*UserHandler)(nil)

type User struct {
	Id        string `json:"id,omitempty" example:"fe9f6053-8929-4868-be47-f3015c46577b" format:"uuid"`
	Name      string `json:"name,omitempty" example:"John Doe"`
	CreatedAt string `json:"created_at,omitempty" example:"2009-11-10T23:00:00Z" format:"RFC3339"`
	UpdatedAt string `json:"updated_at,omitempty" example:"2009-11-10T23:30:00Z" format:"RFC3339"`
}

// UserHandler handles all `/user` endpoints.
type UserHandler struct {
	router      chi.Router
	userService pb.UserServiceClient
	logger      logging.Logger
}

func MakeUserHandler(grpcClient pb.UserServiceClient, translator tokens.Translator, logger logging.Logger) UserHandler {
	s := UserHandler{
		router:      chi.NewRouter(),
		userService: grpcClient,
		logger:      logger,
	}
	s.registerRoutes(translator)
	return s
}

// registerRoutes registers handlers and middleware for each route.
// All routes are registered on '/' so that the handler can be mounted on any path.
func (s UserHandler) registerRoutes(translator tokens.Translator) {
	s.router.With(otelhttp.NewMiddleware("GetUser")).Get("/{id}", httpe.ToHandlerFunc(s.GetUser, s.logger))
	s.router.With(otelhttp.NewMiddleware("GetUsers")).Get("/", httpe.ToHandlerFunc(s.GetUsers, s.logger))
	s.router.With(otelhttp.NewMiddleware("CreateUser")).Post("/", httpe.ToHandlerFunc(s.CreateUser, s.logger))
	s.router.With(otelhttp.NewMiddleware("UpdateUser")).Patch("/{id}", httpe.ToHandlerFunc(middleware.Apply(s.UpdateUser, middleware.Auth(translator)), s.logger))
	s.router.With(otelhttp.NewMiddleware("DeleteUser")).Delete("/{id}", httpe.ToHandlerFunc(middleware.Apply(s.DeleteUser, middleware.Auth(translator)), s.logger))
}

// ServeHTTP is called on each request before it's passed to the handler.
func (s UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetUser retrieves public info of a user with given ID from the UserService.
//
//	@Id			GetUser
//	@Tags		users
//	@Summary	Get a user by ID.
//	@Router		/users/{id}	[get]
//	@Produce	json
//	@Param		id	path		string	true	"User ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Success	200	{object}	User
//	@Failure	404	"User could not be found."
//	@Failure	500	"An unexpected error occurred."
func (s UserHandler) GetUser(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	userId := r.PathValue("id")
	if userId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "user not found")
	}

	resp, err := s.userService.Get(ctx, &pb.GetUserRequest{Id: userId})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return nil, httpe.NewError(http.StatusNotFound, "user not found")
		}
		s.logger.Log(ctx, "Failed to get user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, pbToUser(resp.GetUser())), nil
}

// GetUsers queries users mathing given filter from the UserService.
// Reads filter, pagination offset and limit from URL query params and
// forwards them to the UserService.
//
//	@Id			GetUsers
//	@Tags		users
//	@Summary	Get paginated users.
//	@Router		/users/	[get]
//	@Produce	json
//	@Param		offset	query	int		false	"items offset"			Example(60)
//	@Param		limit	query	int		false	"item limit per page"	Example(30)
//	@Param		filter	query	string	false	"search filter"			Example(name[$eq]=john&email[$eq]=doe@example.com)
//	@Success	200		{array}	User
//	@Failure	500		"An unexpected error occurred."
func (s UserHandler) GetUsers(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	queryParams := r.URL.Query()
	offset := queryParams.Get("offset")
	limit := queryParams.Get("limit")
	filter := queryParams.Get("filter")

	stream, err := s.userService.GetStream(ctx, &pb.GetUsersRequest{Offset: offset, Limit: limit, Filter: filter})
	if err != nil {
		s.logger.Log(ctx, "Failed to init user stream", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	var users []User
	for {
		pbUser, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			s.logger.Log(ctx, "Failed to read user from stream", "transport", "grpc", "err", err)
			return nil, httpe.NewGenericError(http.StatusInternalServerError)
		}
		users = append(users, pbToUser(pbUser))
	}

	return httpe.NewResponse(http.StatusOK, users), nil
}

// CreateUserRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type CreateUserRequest struct {
	Name     string `json:"name,omitempty" example:"username123"`
	Email    string `json:"email,omitempty" example:"example@gmail.com" format:"email"`
	Password string `json:"password,omitempty" example:"zaq1@WSXEDC"`
}

// CreateUserResponse exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type CreateUserResponse struct {
	Id string `json:"id,omitempty" example:"fe9f6053-8929-4868-be47-f3015c46577b" format:"uuid"`
}

// CreateUser creates a user in the UserService and returns its ID.
//
//	@Id			CreateUser
//	@Tags		users
//	@Summary	Create a user.
//	@Router		/users/	[post]
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		CreateUserRequest	true	"Payload"
//	@Success	201		{object}	CreateUserResponse
//	@Failure	400		"The request body could not be parsed."
//	@Failure	500		"An unexpected error occurred."
func (s UserHandler) CreateUser(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	req := CreateUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	user := &pb.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := s.userService.Create(ctx, &pb.CreateUserRequest{User: user})
	if err != nil {
		s.logger.Log(ctx, "Failed to create user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusCreated, CreateUserResponse{Id: resp.Id}), nil
}

// UpdateUserRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type UpdateUserRequest struct {
	Name     string `json:"name,omitempty" example:"username123"`
	Email    string `json:"email,omitempty" example:"example@gmail.com" format:"email"`
	Password string `json:"password,omitempty" example:"zaq1@WSXEDC"`
}

// UpdateUser updates an existing user in the UserService.
//
//	@Id				UpdateUser
//	@Tags			users
//	@Summary		Update a user.
//	@Description	Accepts individual fields to update.
//	@Description	If no fields are provided then the call is a no-op and status 200 is returned.
//	@Router			/users/{id}	[patch]
//	@Security		bearerauth
//	@Accept			json
//	@Param			id		path	string				true	"User ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Param			payload	body	UpdateUserRequest	true	"Payload"
//	@Success		200
//	@Failure		400	"The request body could not be parsed."
//	@Failure		401	"Authorization token is invalid or missing."
//	@Failure		404	"User ID is empty or user does not exist."
//	@Failure		500	"An unexpected error occurred."
func (s UserHandler) UpdateUser(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	ctx, err = middleware.ConvertTokenContext(ctx)
	if err != nil {
		s.logger.Log(ctx, "Failed convert context metadata", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	userId := r.PathValue("id")
	if userId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "user not found")
	}

	req := UpdateUserRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	if req == (UpdateUserRequest{}) {
		// Avoid needless gRPC calls.
		return httpe.NewResponse(http.StatusOK, nil), nil
	}

	user := &pb.User{
		Id:       userId,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if _, err := s.userService.Update(ctx, &pb.UpdateUserRequest{User: user}); err != nil {
		s.logger.Log(ctx, "Failed to update user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusOK, nil), nil
}

// DeleteUser deletes an existing user in the UserService.
//
//	@Id			DeleteUser
//	@Tags		users
//	@Summary	Delete a user.
//	@Router		/users/{id}	[delete]
//	@Security	bearerauth
//	@Param		id	path	string	true	"User ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Success	204
//	@Failure	401	"Authorization token is invalid or missing."
//	@Failure	404	"User could not be found."
//	@Failure	500	"An unexpected error occurred."
func (s UserHandler) DeleteUser(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	ctx, err = middleware.ConvertTokenContext(ctx)
	if err != nil {
		s.logger.Log(ctx, "Failed convert context metadata", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	userId := r.PathValue("id")
	if userId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "user not found")
	}

	if _, err := s.userService.Delete(ctx, &pb.DeleteUserRequest{Id: userId}); err != nil {
		s.logger.Log(ctx, "Failed to delete user", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusNoContent, nil), nil
}

// pbToUser converts pb.User message to an User model.
func pbToUser(v *pb.User) User {
	return User{
		Id:        v.GetId(),
		Name:      v.GetName(),
		CreatedAt: v.GetCreatedAt().AsTime().Format(time.RFC3339),
		UpdatedAt: v.GetUpdatedAt().AsTime().Format(time.RFC3339),
	}
}
