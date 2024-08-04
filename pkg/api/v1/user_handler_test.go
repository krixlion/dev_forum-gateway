package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/dev_forum-auth/pkg/tokens/tokensmocks"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe/middleware"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/nulls"
	"github.com/krixlion/dev_forum-user/pkg/grpc/mocks"
	pb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testtime = time.Now()

func mustMarshalJSON(in any, t *testing.T) []byte {
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatal("Failed to marshal test data to json")
	}
	return out
}

func TestMakeUserHandler(t *testing.T) {
	type args struct {
		grpcClient pb.UserServiceClient
		logger     logging.Logger
	}
	tests := []struct {
		name string
		args args
		want UserHandler
	}{
		{
			name: "Test all deps are assigned and all fields initialized",
			args: args{
				grpcClient: mocks.NewUserClient(),
				logger:     nulls.NullLogger{},
			},
			want: UserHandler{
				router:      chi.NewRouter(),
				userService: mocks.NewUserClient(),
				logger:      nulls.NullLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeUserHandler(tt.args.grpcClient, tokensmocks.NewTokenTranslator(), tt.args.logger)
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(UserHandler{}), cmpopts.IgnoreUnexported(chi.Mux{}, mock.Mock{})) {
				t.Errorf("MakeUserHandler():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_GetUser(t *testing.T) {
	type args struct {
		r *http.Request
	}
	type fields struct {
		grpcClient pb.UserServiceClient
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    httpe.Response
		wantErr bool
	}{
		{
			name: "Test returns user data along with status 200 on valid request with no bearer token",
			fields: fields{
				grpcClient: func() pb.UserServiceClient {
					m := mocks.NewUserClient()
					v := &pb.GetUserResponse{User: &pb.User{
						Id:        "test-id",
						Name:      "test-name",
						Email:     "test-email",
						Password:  "test-password",
						CreatedAt: timestamppb.New(testtime),
						UpdatedAt: timestamppb.New(testtime),
					}}
					m.On("Get", mock.Anything, &pb.GetUserRequest{Id: "test-id"}, mock.AnythingOfType("[]grpc.CallOption")).Return(v, nil).Once()
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
			want: httpe.NewResponse(http.StatusOK, &pb.User{
				Id:        "test-id",
				Name:      "test-name",
				Email:     "test-email",
				Password:  "test-password",
				CreatedAt: timestamppb.New(testtime),
				UpdatedAt: timestamppb.New(testtime),
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

			handler := MakeUserHandler(tt.fields.grpcClient, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})

			got, err := handler.GetUser(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandler.GetUser():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{}), cmpopts.IgnoreUnexported(pb.User{}, timestamppb.Timestamp{})) {
				t.Errorf("UserHandler.GetUser():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_GetUsers(t *testing.T) {
	type fields struct {
		userService pb.UserServiceClient
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
			name: "Test returns users data and status 200 on valid request with no bearer token",
			fields: fields{
				userService: func() pb.UserServiceClient {
					ms := mocks.NewUserStreamClient()
					ms.On("Recv").Return(&pb.User{Id: "test-id"}, nil).Once()
					ms.On("Recv").Return(&pb.User{Id: "test-id2"}, nil).Once()
					ms.On("Recv").Return((*pb.User)(nil), io.EOF).Once()
					m := mocks.NewUserClient()

					m.On("GetStream", mock.Anything, &pb.GetUsersRequest{Offset: "test-offset", Limit: "test-limit", Filter: "test-filter"}, mock.AnythingOfType("[]grpc.CallOption")).Return(ms, nil).Once()
					return m
				}(),
			},
			args: args{
				r: httptest.NewRequest("GET", "/?filter=test-filter&offset=test-offset&limit=test-limit", nil),
			},
			want: httpe.NewResponse(http.StatusOK, []*pb.User{
				{
					Id: "test-id",
				},
				{
					Id: "test-id2",
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

			s := MakeUserHandler(tt.fields.userService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.GetUsers(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandler.GetUsers():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{}), cmpopts.IgnoreUnexported(pb.User{}, timestamppb.Timestamp{})) {
				t.Errorf("UserHandler.GetUsers():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_CreateUser(t *testing.T) {
	type fields struct {
		userService pb.UserServiceClient
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
			name: "Test user ID and status 201 is returned on valid request with no bearer token",
			fields: fields{
				userService: func() pb.UserServiceClient {
					m := mocks.NewUserClient()
					m.On("Create", mock.Anything, &pb.CreateUserRequest{User: &pb.User{Id: "test-id"}}, mock.AnythingOfType("[]grpc.CallOption")).Return(&pb.CreateUserResponse{Id: "test-id"}, nil).Once()
					return m
				}(),
			},
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewReader(mustMarshalJSON(&pb.User{Id: "test-id"}, t))),
			},
			want:    httpe.NewResponse(http.StatusCreated, map[string]string{"id": "test-id"}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.r.Context(), time.Second)
			defer cancel()
			tt.args.r = tt.args.r.WithContext(ctx)

			s := MakeUserHandler(tt.fields.userService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.CreateUser(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandler.CreateUser():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("UserHandler.CreateUser():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	type fields struct {
		userService pb.UserServiceClient
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
				userService: func() pb.UserServiceClient {
					m := mocks.NewUserClient()
					m.On("Update", mock.Anything, &pb.UpdateUserRequest{User: &pb.User{Id: "test-id", Name: "test-name"}}, mock.AnythingOfType("[]grpc.CallOption")).Return(&emptypb.Empty{}, nil).Once()
					return m
				}(),
			},
			args: args{
				r: func() *http.Request {
					v := mustMarshalJSON(&pb.User{Id: "test-id", Name: "test-name"}, t)
					r := httptest.NewRequest("PATCH", "/", bytes.NewReader(v))
					r.SetPathValue("id", "test-id")
					return r.WithContext(context.WithValue(r.Context(), middleware.CtxTokenKey{}, "test-token"))
				}(),
			},
			want:    httpe.NewResponse(http.StatusOK, nil),
			wantErr: false,
		},
		{
			name: "Test returns 404 on missing user ID",
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

			s := MakeUserHandler(tt.fields.userService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.UpdateUser(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandler.UpdateUser():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("UserHandler.UpdateUser():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_DeleteUser(t *testing.T) {
	type fields struct {
		userService pb.UserServiceClient
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
				userService: func() pb.UserServiceClient {
					m := mocks.NewUserClient()
					m.On("Delete", mock.Anything, &pb.DeleteUserRequest{Id: "test-id"}, mock.AnythingOfType("[]grpc.CallOption")).Return(&emptypb.Empty{}, nil).Once()
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
			name: "Test returns 404 on missing user ID",
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

			s := MakeUserHandler(tt.fields.userService, tokensmocks.NewTokenTranslator(), nulls.NullLogger{})
			got, err := s.DeleteUser(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandler.DeleteUser():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(httpe.HttpResponse{})) {
				t.Errorf("UserHandler.DeleteUser():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}
