package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-auth/pkg/tokens/tokensmocks"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
	"github.com/krixlion/dev_forum-lib/nulls"
	"github.com/stretchr/testify/mock"
)

func TestAuth(t *testing.T) {
	t.Run("Test returns 200 on a valid token and that token is extractable form request context", func(t *testing.T) {
		w := httptest.NewRecorder()

		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Add("Authorization", "Bearer test-token")

		wantToken := "test-token-translated"

		m := tokensmocks.NewTokenTranslator()
		m.On("TranslateAccessToken", mock.Anything, "test-token").Return(wantToken, nil).Once()

		h := httpe.HandlerEFunc(func(r *http.Request) (httpe.Response, error) {
			gotToken, ok := r.Context().Value(CtxTokenKey{}).(string)
			if !ok {
				t.Errorf("Auth(): failed to extract token from context")
				return nil, nil
			}

			if gotToken != wantToken {
				t.Errorf("Auth(): handler received an unexpected token:\n got = %v\n want = %v\n", gotToken, wantToken)
				return nil, nil
			}

			return httpe.NewResponse(200, nil), nil
		})

		httpe.ToHandlerFunc(Auth(m)(h), nulls.NullLogger{}).ServeHTTP(w, r)

		res := w.Result()
		defer res.Body.Close()

		got := res.StatusCode
		want := http.StatusOK
		if got != want {
			t.Errorf("Auth():\n got = %v\n want = %v", got, want)
		}
	})

	type args struct {
		translator tokens.Translator
		r          *http.Request
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Test returns 403 on missing authorization header",
			args: args{
				r:          httptest.NewRequest("POST", "/", nil),
				translator: tokensmocks.NewTokenTranslator(),
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Test returns 403 on missing Bearer token",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", nil)
					r.Header.Set("Authorization", "")
					return r
				}(),
				translator: tokensmocks.NewTokenTranslator(),
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Test returns 403 on non-Bearer token",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", nil)
					r.Header.Set("Authorization", "Basic sadfscubjh")
					return r
				}(),
				translator: tokensmocks.NewTokenTranslator(),
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Test returns 403 on invalid token",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", nil)
					r.Header.Set("Authorization", "Bearer sadfscubjh")
					return r
				}(),
				translator: func() tokens.Translator {
					m := tokensmocks.NewTokenTranslator()
					m.On("TranslateAccessToken", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("test-err")).Once()
					return m
				}(),
			},
			want: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handlerStub := httpe.HandlerEFunc(func(r *http.Request) (httpe.Response, error) { return httpe.NewResponse(200, nil), nil })

			httpe.ToHandlerFunc(Auth(tt.args.translator)(handlerStub), nulls.NullLogger{}).ServeHTTP(w, tt.args.r)

			res := w.Result()
			defer res.Body.Close()

			if got := res.StatusCode; got != tt.want {
				t.Errorf("Auth(): unexpected resp body:\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}
