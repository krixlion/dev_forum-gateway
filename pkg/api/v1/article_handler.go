package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	pb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe/middleware"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ http.Handler = (*ArticleHandler)(nil)

// ArticleHandler handles all `/article` endpoints.
type ArticleHandler struct {
	router         chi.Router
	articleService pb.ArticleServiceClient
	logger         logging.Logger
}

func MakeArticleHandler(grpcClient pb.ArticleServiceClient, translator tokens.Translator, logger logging.Logger) ArticleHandler {
	s := ArticleHandler{
		router:         chi.NewRouter(),
		articleService: grpcClient,
		logger:         logger,
	}
	s.registerRoutes(translator)
	return s
}

// registerRoutes registers handlers and middleware for each route.
// All routes are registered on '/' so that the handler can be mounted on any path.
func (s ArticleHandler) registerRoutes(translator tokens.Translator) {
	s.router.With(otelhttp.NewMiddleware("GetArticle")).Get("/{id}", httpe.ToHandlerFunc(s.GetArticle, s.logger))
	s.router.With(otelhttp.NewMiddleware("GetArticles")).Get("/", httpe.ToHandlerFunc(s.GetArticles, s.logger))
	s.router.With(otelhttp.NewMiddleware("CreateArticle")).Post("/", httpe.ToHandlerFunc(middleware.Apply(s.CreateArticle, middleware.Auth(translator)), s.logger))
	s.router.With(otelhttp.NewMiddleware("UpdateArticle")).Patch("/{id}", httpe.ToHandlerFunc(middleware.Apply(s.UpdateArticle, middleware.Auth(translator)), s.logger))
	s.router.With(otelhttp.NewMiddleware("DeleteArticle")).Delete("/{id}", httpe.ToHandlerFunc(middleware.Apply(s.DeleteArticle, middleware.Auth(translator)), s.logger))
}

// ServeHTTP is called on each request before it's passed to the handler.
func (s ArticleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetArticle retrieves an article with given ID from the ArticleService.
//
//	@Id			GetArticle
//	@Tags		articles
//	@Summary	Get an article by ID.
//	@Router		/articles/{id}	[get]
//	@Produce	json
//	@Param		id	path		string	true	"Article ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Success	200	{object}	pb.Article
//	@Failure	404	"Article could not be found."
//	@Failure	500	"An unexpected error occurred."
func (s ArticleHandler) GetArticle(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	articleId := r.PathValue("id")
	if articleId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "article not found")
	}

	resp, err := s.articleService.Get(ctx, &pb.GetArticleRequest{Id: articleId})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return nil, httpe.NewError(http.StatusNotFound, "article not found")
		}
		s.logger.Log(ctx, "Failed to get article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	return httpe.NewResponse(http.StatusOK, resp.GetArticle()), nil
}

// GetArticles queries articles mathing given filter from the ArticleService.
// Reads pagination offset and limit from URL query params and
// forwards them to the ArticleService.
//
//	@Id			GetArticles
//	@Tags		articles
//	@Summary	Get paginated articles.
//	@Router		/articles/	[get]
//	@Produce	json
//	@Param		offset	query	int	false	"items offset"			Example(60)
//	@Param		limit	query	int	false	"item limit per page"	Example(30)
//	@Success	200		{array}	pb.Article
//	@Failure	500		"An unexpected error occurred."
func (s ArticleHandler) GetArticles(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)
	queryParams := r.URL.Query()
	offset := queryParams.Get("offset")
	limit := queryParams.Get("limit")

	stream, err := s.articleService.GetStream(ctx, &pb.GetArticlesRequest{Offset: offset, Limit: limit})
	if err != nil {
		s.logger.Log(ctx, "Failed to init article stream", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	var articles []*pb.Article
	for {
		article, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			s.logger.Log(ctx, "Failed to read article from stream", "transport", "grpc", "err", err)
			return nil, httpe.NewGenericError(http.StatusInternalServerError)
		}
		articles = append(articles, article)
	}

	return httpe.NewResponse(http.StatusOK, articles), nil
}

// CreateArticleRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type CreateArticleRequest struct {
	Title string `json:"title,omitempty" example:"How to train your AI dragon!"`
	Body  string `json:"body,omitempty" example:"Lorem ipsum dolor sit amet, consectetur adipiscing elit."`
}

// CreateArticleResponse exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type CreateArticleResponse struct {
	Id string `json:"id,omitempty" example:"fe9f6053-8929-4868-be47-f3015c46577b"`
}

// CreateArticle creates an article in the ArticleService and returns its ID.
//
//	@Id			CreateArticle
//	@Tags		articles
//	@Summary	Create an article.
//	@Security	bearerauth
//	@Router		/articles/	[post]
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		CreateArticleRequest	true	"Payload"
//	@Success	201		{object}	CreateArticleResponse
//	@Failure	400		"The request body could not be parsed."
//	@Failure	401		"Authorization token is invalid or missing."
//	@Failure	500		"An unexpected error occurred."
func (s ArticleHandler) CreateArticle(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	ctx, err = middleware.ConvertTokenContext(ctx)
	if err != nil {
		s.logger.Log(ctx, "Failed convert context metadata", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	req := CreateArticleRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	article := &pb.Article{
		Body:  req.Body,
		Title: req.Title,
	}

	resp, err := s.articleService.Create(ctx, &pb.CreateArticleRequest{Article: article})
	if err != nil {
		s.logger.Log(ctx, "Failed to create article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusCreated, CreateArticleResponse{Id: resp.GetId()}), nil
}

// UpdateArticleRequest exists mainly for documentation purposes.
// It's parsed by the OpenAPI docs generator.
type UpdateArticleRequest struct {
	Title string `json:"title,omitempty" example:"How to train your AI dragon!"`
	Body  string `json:"body,omitempty" example:"Lorem ipsum dolor sit amet, consectetur adipiscing elit."`
}

// UpdateArticle updates an existing article in the ArticleService.
//
//	@Id				UpdateArticle
//	@Tags			articles
//	@Summary		Update an article.
//	@Description	Accepts individual fields to update.
//	@Description	If no fields are provided then the call is a no-op and status 200 is returned.
//	@Router			/articles/{id}	[patch]
//	@Security		bearerauth
//	@Accept			json
//	@Param			id		path	string					true	"Article ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Param			payload	body	UpdateArticleRequest	true	"Payload"
//	@Success		200
//	@Failure		400	"The request body could not be parsed or the ID was not given."
//	@Failure		401	"Authorization token is invalid or missing."
//	@Failure		500	"An unexpected error occurred."
func (s ArticleHandler) UpdateArticle(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	ctx, err = middleware.ConvertTokenContext(ctx)
	if err != nil {
		s.logger.Log(ctx, "Failed convert context metadata", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	articleId := r.PathValue("id")
	if articleId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "article not found")
	}

	req := UpdateArticleRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	if req == (UpdateArticleRequest{}) {
		// Avoid needless gRPC calls.
		return httpe.NewResponse(http.StatusOK, nil), nil
	}

	article := &pb.Article{
		Id:    articleId,
		Title: req.Title,
		Body:  req.Body,
	}

	if _, err := s.articleService.Update(ctx, &pb.UpdateArticleRequest{Article: article}); err != nil {
		s.logger.Log(ctx, "Failed to update article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusOK, nil), nil
}

// DeleteArticle deletes an existing article in the ArticleService.
//
//	@Id			DeleteArticle
//	@Tags		articles
//	@Summary	Delete an article.
//	@Router		/articles/{id}	[delete]
//	@Security	bearerauth
//	@Param		id	path	string	true	"Article ID"	Format(uuid)	Example(fe9f6053-8929-4868-be47-f3015c46577b)
//	@Success	204
//	@Failure	401	"Authorization token is invalid or missing."
//	@Failure	404	"Article could not be found."
//	@Failure	500	"An unexpected error occurred."
func (s ArticleHandler) DeleteArticle(r *http.Request) (_ httpe.Response, err error) {
	ctx := r.Context()
	defer tracing.SetSpanErr(trace.SpanFromContext(ctx), err)

	ctx, err = middleware.ConvertTokenContext(ctx)
	if err != nil {
		s.logger.Log(ctx, "Failed convert context metadata", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}

	articleId := r.PathValue("id")
	if articleId == "" {
		return nil, httpe.NewError(http.StatusNotFound, "article not found")
	}

	if _, err := s.articleService.Delete(ctx, &pb.DeleteArticleRequest{Id: articleId}); err != nil {
		s.logger.Log(ctx, "Failed to delete article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusNoContent, nil), nil
}
