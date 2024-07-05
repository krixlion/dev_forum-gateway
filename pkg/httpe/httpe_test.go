package httpe

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/nulls"
)

func TestNewHandler(t *testing.T) {
	type args struct {
		fn     HandlerEFunc
		logger logging.Logger
		r      *http.Request
	}
	type result struct {
		statusCode int
		body       string
		headers    map[string]string
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "Test handler sends unchanged response on no error",
			args: args{
				r: httptest.NewRequest("POST", "/", nil),
				fn: func(r *http.Request) (Response, error) {
					return NewResponse(http.StatusOK, map[string]string{"testdata": "test"}), nil
				},
				logger: nulls.NullLogger{},
			},
			want: result{
				statusCode: http.StatusOK,
				body:       `{"testdata":"test"}` + "\n",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			name: "Test handler sends unchanged error response on HTTP error",
			args: args{
				r: httptest.NewRequest("POST", "/", nil),
				fn: func(r *http.Request) (Response, error) {
					return nil, HttpError{
						status: http.StatusBadRequest,
						msg:    "test err msg",
					}
				},
				logger: nulls.NullLogger{},
			},
			want: result{
				statusCode: http.StatusBadRequest,
				body:       `{"error":"test err msg"}` + "\n",
				headers: map[string]string{
					"Content-Type":           "application/json",
					"X-Content-Type-Options": "nosniff",
				},
			},
		},
		{
			name: "Test handler sends generic 500 response on non HTTP errors",
			args: args{
				r: httptest.NewRequest("POST", "/", nil),
				fn: func(r *http.Request) (Response, error) {
					return nil, errors.New("test err")
				},
				logger: nulls.NullLogger{},
			},
			want: result{
				statusCode: http.StatusInternalServerError,
				body:       `{"error":"Internal Server Error"}` + "\n",
				headers: map[string]string{
					"Content-Type":           "application/json",
					"X-Content-Type-Options": "nosniff",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			NewHandler(tt.args.fn, tt.args.logger).ServeHTTP(w, tt.args.r)

			got := w.Result()
			if got.StatusCode != tt.want.statusCode {
				t.Errorf("NewHandler.ServeHTTP(): unexpected status code:\n got = %v\n want = %v", got.StatusCode, tt.want.statusCode)
			}

			for key, want := range tt.want.headers {
				if got := got.Header.Get(key); got != want {
					t.Errorf("NewHandler.ServeHTTP(): unexpected header:\n got = %v\n want = %v\n", got, want)
				}
			}

			body, err := io.ReadAll(got.Body)
			if err != nil {
				t.Fatalf("NewHandler.ServeHTTP():\n failed to read response body = %v\n", err)
			}

			if v := string(body); v != tt.want.body {
				t.Errorf("NewHandler.ServeHTTP():\n got = %v\n want = %v", v, tt.want.body)
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	type args struct {
		status int
		body   interface{}
	}
	tests := []struct {
		name string
		args args
		want Response
	}{
		{
			name: "Test all fields are assigned and initialized",
			args: args{
				status: http.StatusOK,
				body:   "test-body",
			},
			want: HttpResponse{
				status: http.StatusOK,
				body:   "test-body",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewResponse(tt.args.status, tt.args.body); !cmp.Equal(got, tt.want, cmp.AllowUnexported(HttpResponse{})) {
				t.Errorf("NewResponse():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestStdResponse_Status(t *testing.T) {
	type fields struct {
		status int
		body   interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "Test status is returned unchanged",
			fields: fields{
				status: http.StatusOK,
				body:   "test-body",
			},
			want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := HttpResponse{
				status: tt.fields.status,
				body:   tt.fields.body,
			}
			if got := resp.Status(); got != tt.want {
				t.Errorf("StdResponse.Status():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestStdResponse_Body(t *testing.T) {
	type fields struct {
		status int
		body   interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{
			name: "Test body is returned unchanged",
			fields: fields{
				status: http.StatusOK,
				body:   "test-body",
			},
			want: "test-body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := HttpResponse{
				status: tt.fields.status,
				body:   tt.fields.body,
			}
			if got := resp.Body(); !cmp.Equal(got, tt.want, cmp.AllowUnexported(HttpResponse{})) {
				t.Errorf("StdResponse.Body():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestHttpError_Error(t *testing.T) {
	type fields struct {
		status int
		msg    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test error message is returned unchanged",
			fields: fields{
				status: http.StatusNotFound,
				msg:    "Not Found",
			},
			want: "Not Found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := HttpError{
				status: tt.fields.status,
				msg:    tt.fields.msg,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("HttpError.Error():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestHttpError_MarshalJSON(t *testing.T) {
	type fields struct {
		status int
		msg    string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "Test error message is properly encoded",
			fields: fields{
				status: http.StatusOK,
				msg:    "OK",
			},
			want: []byte(`{"error":"OK"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := HttpError{
				status: tt.fields.status,
				msg:    tt.fields.msg,
			}
			got, err := e.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("HttpError.MarshalJSON():\n error = %v\n wantErr = %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("HttpError.MarshalJSON():\n got = %v\n want = %v", got, tt.want)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	type args struct {
		status int
		msg    string
	}
	tests := []struct {
		name string
		args args
		want HttpError
	}{
		{
			name: "Test all fields are assigned and initialized",
			args: args{
				status: http.StatusNotFound,
				msg:    "Not Found",
			},
			want: HttpError{
				status: http.StatusNotFound,
				msg:    "Not Found",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewError(tt.args.status, tt.args.msg); !cmp.Equal(err, tt.want, cmp.AllowUnexported(HttpError{})) {
				t.Errorf("NewError():\n error = %v\n want = %v", err, tt.want)
			}
		})
	}
}

func Test_respondError(t *testing.T) {
	type args struct {
		err HttpError
	}
	type result struct {
		statusCode int
		body       string
	}
	tests := []struct {
		name    string
		args    args
		want    result
		wantErr bool
	}{
		{
			name: "Test err response is sent with no errors",
			args: args{
				err: HttpError{
					status: http.StatusInternalServerError,
					msg:    "InternalServerError",
				},
			},
			want: result{
				statusCode: http.StatusInternalServerError,
				body:       `{"error":"InternalServerError"}` + "\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			if err := respondError(w, tt.args.err); (err != nil) != tt.wantErr {
				t.Fatalf("respondError():\n error = %v\n wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			got := w.Result()
			if got.StatusCode != tt.want.statusCode {
				t.Errorf("respondError(): unexpected status code:\n got = %v\n want = %v", got.StatusCode, tt.want.statusCode)
			}

			headers := map[string]string{
				"Content-Type":           "application/json",
				"X-Content-Type-Options": "nosniff",
			}

			for key, want := range headers {
				if got := got.Header.Get(key); got != want {
					t.Errorf("respondError(): unexpected header:\n got = %v\n want = %v\n", got, want)
				}
			}

			body, err := io.ReadAll(got.Body)
			if err != nil {
				t.Fatalf("respondError():\n failed to read response body = %v\n", err)
			}

			if v := string(body); v != tt.want.body {
				t.Errorf("respondError():\n got = %v\n want = %v", v, tt.want.body)
			}
		})
	}
}

func Test_respond(t *testing.T) {
	type result struct {
		statusCode int
		body       string
	}
	tests := []struct {
		name    string
		resp    Response
		want    result
		wantErr bool
	}{
		{
			name: "Test empty response with status 200 is sent with no errors",
			resp: NewResponse(http.StatusOK, nil),
			want: result{
				statusCode: http.StatusOK,
				body:       "null\n",
			},
			wantErr: false,
		},
		{
			name: "Test encodes json response with no errors",
			resp: NewResponse(http.StatusCreated, map[string]string{
				"data": "testdata",
			}),
			want: result{
				statusCode: http.StatusCreated,
				body:       `{"data":"testdata"}` + "\n",
			},
			wantErr: false,
		},
		{
			name: "Test returns an error when trying to encode invalid input to json",
			resp: NewResponse(http.StatusCreated, make(chan int)),
			want: result{
				statusCode: http.StatusCreated,
				body:       `{"data":"testdata"}` + "\n",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			if err := respond(w, tt.resp); (err != nil) != tt.wantErr {
				t.Fatalf("respond():\n error = %v\n wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			got := w.Result()
			if got.StatusCode != tt.want.statusCode {
				t.Errorf("respond(): unexpected status code:\n got = %v\n want = %v", got.StatusCode, tt.want.statusCode)
			}

			if v := got.Header.Get("Content-Type"); v != "application/json" {
				t.Errorf("respond(): unexpected header:\n got = %v\n want = application/json", v)
			}

			body, err := io.ReadAll(got.Body)
			if err != nil {
				t.Fatalf("respond():\n failed to read response body = %v\n", err)
			}

			if v := string(body); v != tt.want.body {
				t.Errorf("respond():\n got = %v\n want = %v", v, tt.want.body)
			}
		})
	}
}

func TestNewGenericError(t *testing.T) {
	type args struct {
		status int
	}
	tests := []struct {
		name string
		args args
		want HttpError
	}{
		{
			name: "Test all fields are assigned and initialized",
			args: args{
				status: http.StatusNotFound,
			},
			want: HttpError{
				status: http.StatusNotFound,
				msg:    "Not Found",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGenericError(tt.args.status); !cmp.Equal(got, tt.want, cmp.AllowUnexported(HttpError{})) {
				t.Errorf("NewGenericError():\n got = %v\n want = %v\n", got, tt.want)
			}
		})
	}
}
