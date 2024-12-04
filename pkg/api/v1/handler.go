package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	articlepb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	authpb "github.com/krixlion/dev_forum-auth/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-lib/logging"
	userpb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
)

// NewHandler returns a handler for the gateway REST API.
//
//	@Title							Dev-Forum
//	@Version						1.0
//	@Description					API for interacting with the forum.
//	@Securitydefinitions.BearerAuth	bearerauth
func NewHandler(authClient authpb.AuthServiceClient, userClient userpb.UserServiceClient, articleClient articlepb.ArticleServiceClient, tr tokens.Translator, l logging.Logger) http.Handler {
	router := chi.NewRouter()
	router.Mount("/articles", MakeArticleHandler(articleClient, tr, l))
	router.Mount("/users", MakeUserHandler(userClient, tr, l))
	router.Mount("/auth", MakeAuthHandler(authClient, l))
	return router
}
