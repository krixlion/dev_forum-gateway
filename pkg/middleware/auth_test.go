package middleware

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krixlion/dev_forum-auth/pkg/tokens"
	"github.com/krixlion/dev_forum-auth/pkg/tokens/tokensmocks"
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
		m.On("TranslateAccessToken", "test-token").Return(wantToken, nil).Once()

		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotToken, ok := r.Context().Value(CtxTokenKey{}).(string)
			if !ok {
				t.Errorf("Auth(): failed to extract token from context")
				return
			}
			if gotToken != wantToken {
				t.Errorf("Auth(): handler received an unexpected token:\n got = %v\n want = %v\n", gotToken, wantToken)
				return
			}
			if _, err := w.Write(nil); err != nil {
				log.Fatalf("Auth(): failed to write nil resp from stub handler: %v", err)
			}
		})

		Auth(m, nulls.NullLogger{})(h).ServeHTTP(w, r)

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
					m.On("TranslateAccessToken", mock.AnythingOfType("string")).Return("", errors.New("test-err")).Once()
					return m
				}(),
			},
			want: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handlerStub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write(nil); err != nil {
					log.Fatalf("Auth(): failed to write nil resp from stub handler: %v", err)
				}
			})

			Auth(tt.args.translator, nulls.NullLogger{})(handlerStub).ServeHTTP(w, tt.args.r)

			res := w.Result()
			defer res.Body.Close()

			if got := res.StatusCode; got != tt.want {
				t.Errorf("Auth(): unexpected resp body:\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}
