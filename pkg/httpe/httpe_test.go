package httpe

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krixlion/dev_forum-lib/logging"
)

func TestNewHandlerFunc(t *testing.T) {
	type args struct {
		fn     HandlerEFunc
		logger logging.Logger
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHandlerFunc(tt.args.fn, tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHandlerFunc():\n got = %v\n want = %v", got, tt.want)
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
		w   http.ResponseWriter
		err HttpError
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := respondError(tt.args.w, tt.args.err); (err != nil) != tt.wantErr {
				t.Errorf("respondError():\n error = %v\n wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func Test_respond(t *testing.T) {
	type args struct {
		w    http.ResponseWriter
		resp Response
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := respond(tt.args.w, tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("respond():\n error = %v\n wantErr = %v", err, tt.wantErr)
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
