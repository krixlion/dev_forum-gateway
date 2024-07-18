package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	pb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ArticleHandler struct {
	router         chi.Router
	articleService pb.ArticleServiceClient
	logger         logging.Logger
}

func MakeArticleHandler(grpcClient pb.ArticleServiceClient, logger logging.Logger) ArticleHandler {
	s := ArticleHandler{
		router:         chi.NewRouter(),
		articleService: grpcClient,
		logger:         logger,
	}
	s.registerRoutes()
	return s
}

func (s ArticleHandler) registerRoutes() {
	s.router.Get("/{id}", otelhttp.NewHandler(httpe.NewHandler(s.GetArticle, s.logger), "GetArticle").ServeHTTP)
	s.router.Get("/", otelhttp.NewHandler(httpe.NewHandler(s.GetArticles, s.logger), "GetArticles").ServeHTTP)
	s.router.Post("/", otelhttp.NewHandler(httpe.NewHandler(s.CreateArticle, s.logger), "CreateArticle").ServeHTTP)
	s.router.Patch("/{id}", otelhttp.NewHandler(httpe.NewHandler(s.UpdateArticle, s.logger), "UpdateArticle").ServeHTTP)
	s.router.Delete("/{id}", otelhttp.NewHandler(httpe.NewHandler(s.DeleteArticle, s.logger), "DeleteArticle").ServeHTTP)
}

func (s ArticleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetArticle retrieves an article with given ID from the ArticleService.
//   - Returns 200 if no error is encountered.
//   - Returns 404 if the ID is an empty string or the service returned code NotFound.
//   - Returns 500 on any error.
func (s ArticleHandler) GetArticle(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	articleId := r.PathValue("id")
	if errMsg := "article not found"; articleId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	resp, err := s.articleService.Get(ctx, &pb.GetArticleRequest{Id: articleId})
	if err != nil {
		tracing.SetSpanErr(span, err)
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
//   - Returns 200 if no error is encountered.
//   - Returns 500 on any error.
func (s ArticleHandler) GetArticles(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	queryParams := r.URL.Query()
	offset := queryParams.Get("offset")
	limit := queryParams.Get("limit")

	stream, err := s.articleService.GetStream(ctx, &pb.GetArticlesRequest{Offset: offset, Limit: limit})
	if err != nil {
		tracing.SetSpanErr(span, err)
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
			tracing.SetSpanErr(span, err)
			s.logger.Log(ctx, "Failed to read article from stream", "transport", "grpc", "err", err)
			return nil, httpe.NewGenericError(http.StatusInternalServerError)
		}
		articles = append(articles, article)
	}

	return httpe.NewResponse(http.StatusOK, articles), nil
}

// CreateArticle creates an article in the ArticleService.
//   - Returns 201 if no error is encountered. Response contains ID of created article.
//   - Returns 400 when an error is encountered when decoding request's body.
//   - Returns 500 on any other error.
func (s ArticleHandler) CreateArticle(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	article := &pb.Article{}
	if err := json.NewDecoder(r.Body).Decode(article); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	resp, err := s.articleService.Create(ctx, &pb.CreateArticleRequest{Article: article})
	if err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to create article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusCreated, map[string]string{"id": resp.Id}), nil
}

// UpdateArticle updates an existing article in the ArticleService.
//   - Returns 200 if no error is encountered.
//   - Returns 404 if the ID is an empty string.
//   - Returns 400 when an error is encountered when decoding request's body.
//   - Returns 500 on any other error.
func (s ArticleHandler) UpdateArticle(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	articleId := r.PathValue("id")
	if errMsg := "article not found"; articleId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	article := &pb.Article{}
	if err := json.NewDecoder(r.Body).Decode(article); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to decode request json body", "transport", "http", "err", err)
		return nil, httpe.NewGenericError(http.StatusBadRequest)
	}

	article.Id = articleId

	if _, err := s.articleService.Update(ctx, &pb.UpdateArticleRequest{Article: article}); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to update article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusOK, nil), nil
}

// DeleteArticle deletes an existing article in the ArticleService.
//   - Returns 204 if no error is encountered.
//   - Returns 404 if the ID is an empty string.
//   - Returns 500 on any other error.
func (s ArticleHandler) DeleteArticle(r *http.Request) (httpe.Response, error) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	articleId := r.PathValue("id")
	if errMsg := "article not found"; articleId == "" {
		tracing.SetSpanErr(span, errors.New(errMsg))
		return nil, httpe.NewError(http.StatusNotFound, errMsg)
	}

	if _, err := s.articleService.Delete(ctx, &pb.DeleteArticleRequest{Id: articleId}); err != nil {
		tracing.SetSpanErr(span, err)
		s.logger.Log(ctx, "Failed to delete article", "transport", "grpc", "err", err)
		return nil, httpe.NewGenericError(http.StatusInternalServerError)
	}
	return httpe.NewResponse(http.StatusNoContent, nil), nil
}
