package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	pb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
	"google.golang.org/grpc"
)

type UserHandler struct {
	router     chi.Router
	grpcClient pb.UserServiceClient
}

func MakeUserHandler(grpcConn *grpc.ClientConn) UserHandler {
	s := UserHandler{
		router:     chi.NewRouter(),
		grpcClient: pb.NewUserServiceClient(grpcConn),
	}
	s.registerRoutes()
	return s
}

func (s UserHandler) registerRoutes() {
	s.router.Get("/{id}", s.GetUser)
	s.router.Get("/", s.GetUsers)
	s.router.Post("/", s.CreateUser)
	s.router.Put("/{id}", s.UpdateUser)
	s.router.Delete("/{id}", s.DeleteUser)
}

func (s UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {}

func (s UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
