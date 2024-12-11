package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/dev_forum-article/pkg/grpc/mocks"
	pb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-auth/pkg/tokens/tokensmocks"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe/middleware"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/nulls"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMakeArticleHandler(t *testing.T) {
	type args struct {
		grpcClient pb.ArticleServiceClient
		logger     logging.Logger
	}
	tests := []struct {
		name string
		args args
		want ArticleHandler
	}{
		{
			name: "Test all deps are assigned and all fields initialized",
			args: args{
				grpcClient: mocks.NewArticleClient(),
				logger:     nulls.NullLogger{},
			},
			want: ArticleHandler{
				router:         chi.NewRouter(),
				articleService: mocks.NewArticleClient(),
				logger:         nulls.NullLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeArticleHandler(tt.args.grpcClient, tokensmocks.NewTokenTranslator(), tt.args.logger)
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(ArticleHandler{}), cmpopts.IgnoreUnexported(chi.Mux{}, mock.Mock{})) {
				t.Errorf("MakeArticleHandler():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestArticleHandler_GetArticle(t *testing.T) {
	type fields struct {
		articleService pb.ArticleServiceClient
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test returns article along with status 200 on valid request with no bearer token",
			fields: fields{
				articleService: func() pb.ArticleServiceClient {
					m := mocks.NewArticleClient()
					v := &pb.GetArticleResponse{Article: &pb.Article{
						Id:        "test-id",
						Title:     "test-title",
						CreatedAt: timestamppb.New(testtime),
						UpdatedAt: timestamppb.New(testtime),
					}}
					m.On("Get", mock.Anything, &pb.GetArticleRequest{Id: "test-id"}, mock.AnythingOfType("[]grpc.CallOption")).Return(v, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("GET", "/", nil)
					r.SetPathValue("id", "test-id")
					return r
				}(),
			},
			want: httpe.NewResponse(http.StatusOK, Article{
				Id:        "test-id",
				Title:     "test-title",
				CreatedAt: testtime.Format(time.RFC3339),
				UpdatedAt: testtime.Format(time.RFC3339),
			}),
			wantErr: false,
		},
		{
			name: "Test returns 404 on missing user ID",
			args: args{
				r: httptest.NewRequest("GET", "/", nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			handler := MakeArticleHandler(tt.fields.articleService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})

			got, err := handler.GetArticle(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArticleHandler.GetArticle():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{}), cmpopts.IgnoreUnexported(pb.Article{}, timestamppb.Timestamp{})) {
				t.Errorf("ArticleHandler.GetArticle():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestArticleHandler_GetArticles(t *testing.T) {
	type fields struct {
		articleService pb.ArticleServiceClient
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test articles and status 200 is returned on valid request with no bearer token",
			fields: fields{
				articleService: func() pb.ArticleServiceClient {
					ms := mocks.NewArticleStreamClient()
					ms.On("Recv").Return(&pb.Article{Id: "test-id", CreatedAt: timestamppb.New(testtime), UpdatedAt: timestamppb.New(testtime)}, nil).Once()
					ms.On("Recv").Return(&pb.Article{Id: "test-id2", CreatedAt: timestamppb.New(testtime), UpdatedAt: timestamppb.New(testtime)}, nil).Once()
					ms.On("Recv").Return((*pb.Article)(nil), io.EOF).Once()
					m := mocks.NewArticleClient()

					m.On("GetStream", mock.Anything, &pb.GetArticlesRequest{Offset: "test-offset", Limit: "test-limit"}, mock.AnythingOfType("[]grpc.CallOption")).Return(ms, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("GET", "/?filter=test-filter&offset=test-offset&limit=test-limit", nil)
					return r
				}(),
			},
			want: httpe.NewResponse(http.StatusOK, []Article{
				{
					Id:        "test-id",
					CreatedAt: testtime.Format(time.RFC3339),
					UpdatedAt: testtime.Format(time.RFC3339),
				},
				{
					Id:        "test-id2",
					CreatedAt: testtime.Format(time.RFC3339),
					UpdatedAt: testtime.Format(time.RFC3339),
				},
			}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeArticleHandler(tt.fields.articleService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.GetArticles(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArticleHandler.GetArticles():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{}), cmpopts.IgnoreUnexported(pb.Article{}, timestamppb.Timestamp{})) {
				t.Errorf("ArticleHandler.GetArticles():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestArticleHandler_CreateArticle(t *testing.T) {
	type fields struct {
		articleService pb.ArticleServiceClient
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test status 201 and ID are returned on success",
			fields: fields{
				articleService: func() pb.ArticleServiceClient {
					m := mocks.NewArticleClient()
					m.On("Create", mock.Anything, &pb.CreateArticleRequest{Article: &pb.Article{Title: "test-title"}}, mock.AnythingOfType("[]grpc.CallOption")).
						Return(&pb.CreateArticleResponse{Id: "test-id"}, nil).
						Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"test-title"}`))
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			want:    httpe.NewResponse(http.StatusCreated, CreateArticleResponse{Id: "test-id"}),
			wantErr: false,
		},
		{
			name: "Test returns 500 error on missing bearer token in request context",
			args: args{
				r: httptest.NewRequest("POST", "/", nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeArticleHandler(tt.fields.articleService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.CreateArticle(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArticleHandler.CreateArticle():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("ArticleHandler.CreateArticle():\n got = %+v\n want = %+v", got, tt.want)
			}
		})
	}
}

func TestArticleHandler_UpdateArticle(t *testing.T) {
	type fields struct {
		articleService pb.ArticleServiceClient
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test status 200 is returned on success",
			fields: fields{
				articleService: func() pb.ArticleServiceClient {
					m := mocks.NewArticleClient()
					m.On("Update", mock.Anything, &pb.UpdateArticleRequest{Article: &pb.Article{Id: "test-id", Title: "test-title"}}, mock.AnythingOfType("[]grpc.CallOption")).Return(&emptypb.Empty{}, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"id":"test-id","title":"test-title"}`))
					r.SetPathValue("id", "test-id")
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			want:    httpe.NewResponse(http.StatusOK, nil),
			wantErr: false,
		},
		{
			name: "Test returns 404 error on missing article ID in path",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("PATCH", "/", nil)
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test returns 500 error on missing bearer token in request context",
			args: args{
				r: httptest.NewRequest("PATCH", "/", nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()

			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeArticleHandler(tt.fields.articleService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.UpdateArticle(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArticleHandler.UpdateArticle():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("ArticleHandler.UpdateArticle():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestArticleHandler_DeleteArticle(t *testing.T) {
	type fields struct {
		articleService pb.ArticleServiceClient
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test status 204 is returned on success",
			fields: fields{
				articleService: func() pb.ArticleServiceClient {
					m := mocks.NewArticleClient()
					m.On("Delete", mock.Anything, &pb.DeleteArticleRequest{Id: "test-id"}, mock.AnythingOfType("[]grpc.CallOption")).Return(&emptypb.Empty{}, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("DELETE", "/", nil)
					r.SetPathValue("id", "test-id")
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			want:    httpe.NewResponse(http.StatusNoContent, nil),
			wantErr: false,
		},
		{
			name: "Test returns 404 on missing article ID in path",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("DELETE", "/", nil)
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test returns 500 error on missing bearer token in request context",
			args: args{
				r: httptest.NewRequest("DELETE", "/", nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeArticleHandler(tt.fields.articleService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.DeleteArticle(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArticleHandler.DeleteArticle():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("ArticleHandler.DeleteArticle():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func Test_pbToArticle(t *testing.T) {
	type args struct {
		v *pb.Article
	}
	tests := []struct {
		name string
		args args
		want Article
	}{
		{
			name: "Test simple message is converted as expected",
			args: args{
				v: &pb.Article{
					Id:        "test-id",
					UserId:    "test-user-id",
					Title:     "test-title",
					Body:      "test-body",
					CreatedAt: timestamppb.New(testtime),
					UpdatedAt: timestamppb.New(testtime),
				},
			},
			want: Article{
				Id:        "test-id",
				UserId:    "test-user-id",
				Title:     "test-title",
				Body:      "test-body",
				CreatedAt: testtime.Format(time.RFC3339),
				UpdatedAt: testtime.Format(time.RFC3339),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pbToArticle(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pbToArticle():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}
