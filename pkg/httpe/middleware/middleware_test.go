package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krixlion/dev_forum-gateway/pkg/httpe"
)

func TestApply(t *testing.T) {
	t.Run("Test middleware is applied in the same order as provided", func(t *testing.T) {
		got := []string{}
		want := []string{"1", "2", "3"}
		stub := func(r *http.Request) (httpe.Response, error) { return httpe.NewResponse(200, nil), nil }

		m1 := func(httpe.HandlerEFunc) httpe.HandlerEFunc {
			got = append(got, "1")
			return stub
		}

		m2 := func(httpe.HandlerEFunc) httpe.HandlerEFunc {
			got = append(got, "2")
			return stub
		}

		m3 := func(httpe.HandlerEFunc) httpe.HandlerEFunc {
			got = append(got, "3")
			return stub
		}

		h := Apply(httpe.HandlerEFunc(stub), m1, m2, m3)

		h(httptest.NewRequest("GET", "/", nil)) //nolint:errcheck // return vals are irrelevant

		if !cmp.Equal(got, want) {
			t.Errorf("Apply():\n got = %v\n want = %v", got, want)
		}
	})
}
