package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/dev_forum-auth/pkg/grpc/mocks"
	pb "github.com/krixlion/dev_forum-auth/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/nulls"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestMakeAuthHandler(t *testing.T) {
	type args struct {
		grpcClient pb.AuthServiceClient
		logger     logging.Logger
	}
	tests := []struct {
		name string
		args args
		want AuthHandler
	}{
		{
			name: "Test all deps are assigned and all fields initialized",
			args: args{
				grpcClient: mocks.NewAuthClient(),
				logger:     nulls.NullLogger{},
			},
			want: AuthHandler{
				router:      chi.NewRouter(),
				authService: mocks.NewAuthClient(),
				logger:      nulls.NullLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeAuthHandler(tt.args.grpcClient, tt.args.logger)
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(AuthHandler{}), cmpopts.IgnoreUnexported(chi.Mux{}, mock.Mock{})) {
				t.Errorf("MakeAuthHandler():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestAuthHandler_SignIn(t *testing.T) {
	type fields struct {
		authService pb.AuthServiceClient
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
			name: "Test status 200 and refresh token are returned on success",
			fields: fields{
				authService: func() pb.AuthServiceClient {
					m := mocks.NewAuthClient()
					m.On("SignIn", mock.Anything, &pb.SignInRequest{Password: "test-password", Email: "test@test.test"}, mock.AnythingOfType("[]grpc.CallOption")).
						Return(&pb.SignInResponse{RefreshToken: "test-refresh-token"}, nil).
						Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"password":"test-password","email":"test@test.test"}`))
					return r
				}(),
			},
			want:    httpe.NewResponse(http.StatusOK, SignInResponse{RefreshToken: "test-refresh-token"}),
			wantErr: false,
		},
		{
			name: "Test returns 400 error on missing password",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"test@test.test"}`))
					return r
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test returns 400 error on missing email",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"password":"test-password"}`))
					return r
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeAuthHandler(tt.fields.authService, nulls.NullLogger{})
			got, err := s.SignIn(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthHandler.SignIn():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("AuthHandler.SignIn():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestAuthHandler_SignOut(t *testing.T) {
	type fields struct {
		authService pb.AuthServiceClient
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
				authService: func() pb.AuthServiceClient {
					m := mocks.NewAuthClient()
					m.On("SignOut", mock.Anything, &pb.SignOutRequest{RefreshToken: "test-refresh-token"}, mock.AnythingOfType("[]grpc.CallOption")).Return(&emptypb.Empty{}, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"refresh_token":"test-refresh-token"}`))
					return r
				}(),
			},
			want:    httpe.NewResponse(http.StatusOK, nil),
			wantErr: false,
		},
		{
			name: "Test returns 400 error on missing refresh token",
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

			s := MakeAuthHandler(tt.fields.authService, nulls.NullLogger{})
			got, err := s.SignOut(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthHandler.SignOut():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("AuthHandler.SignOut():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestAuthHandler_GetAccessToken(t *testing.T) {
	type fields struct {
		authService pb.AuthServiceClient
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
			name: "Test status 200 and access token are returned on success",
			fields: fields{
				authService: func() pb.AuthServiceClient {
					m := mocks.NewAuthClient()
					m.On("GetAccessToken", mock.Anything, &pb.GetAccessTokenRequest{RefreshToken: "test-refresh-token"}, mock.AnythingOfType("[]grpc.CallOption")).
						Return(&pb.GetAccessTokenResponse{AccessToken: "test-access-token"}, nil).
						Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", strings.NewReader(`{"refresh_token":"test-refresh-token"}`))
					return r
				}(),
			},
			want:    httpe.NewResponse(http.StatusOK, GetAccessTokenResponse{AccessToken: "test-access-token"}),
			wantErr: false,
		},
		{
			name: "Test returns 400 on missing refresh token",
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

			s := MakeAuthHandler(tt.fields.authService, nulls.NullLogger{})
			got, err := s.GetAccessToken(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthHandler.GetAccessToken():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("AuthHandler.GetAccessToken():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}
