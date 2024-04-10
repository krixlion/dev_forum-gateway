package api

import (
	"net/http"

	"github.com/go-chi/chi"
	pb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	"google.golang.org/grpc"
)

type ArticleHandler struct {
	router     chi.Router
	grpcClient pb.ArticleServiceClient
}

func MakeArticleHandler(grpcConn *grpc.ClientConn) ArticleHandler {
	s := ArticleHandler{
		router:     chi.NewRouter(),
		grpcClient: pb.NewArticleServiceClient(grpcConn),
	}
	s.registerRoutes()
	return s
}

func (s ArticleHandler) registerRoutes() {
	s.router.Get("/{id}", s.GetArticle)
	s.router.Get("/", s.GetArticles)
	s.router.Post("/", s.CreateArticle)
	s.router.Put("/{id}", s.UpdateArticle)
	s.router.Delete("/{id}", s.DeleteArticle)
}

func (s ArticleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s ArticleHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s ArticleHandler) GetArticles(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s ArticleHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s ArticleHandler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (s ArticleHandler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
