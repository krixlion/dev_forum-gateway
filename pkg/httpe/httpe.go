package httpe

import (
	"encoding/json"
	"net/http"

	"github.com/krixlion/dev_forum-lib/logging"
)

type HandlerEFunc func(r *http.Request) (Response, error)

// NewHandlerFunc wraps the given [HandlerEFunc] and converts it into a http.HandlerFunc.
// If the [HandlerEFunc] returns a non-nil error other than [HttpError], the handler
// will respond to the request with a internal server error.If the error is a [HttpError], the
// handler will respond with error's status and err message encoded to JSON. If the handlers
// fails to send the response it will abort any further retries and log the error that caused the failure.
func NewHandlerFunc(fn HandlerEFunc, logger logging.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		if err != nil {
			if httpErr, ok := err.(HttpError); ok {
				if err := respondError(w, httpErr); err != nil {
					logger.Log(r.Context(), "Failed to send error response", "transport", "http", "err", err)
				}
				return
			}

			if err := respondError(w, HttpError{status: http.StatusInternalServerError, msg: "Internal Server Error"}); err != nil {
				logger.Log(r.Context(), "Failed to send error response", "transport", "http", "err", err)
			}
			return
		}

		if err := respond(w, resp); err != nil {
			logger.Log(r.Context(), "Failed to send response", "transport", "http", "err", err)
		}
	})
}

// Response represents any non-error HTTP response.
// To respond with an error refer to [HttpError].
// Use [NewResponse] to construct a new response.
type Response interface {
	Status() int
	Body() interface{}
}

// NewResponse returns a new HttpResponse implementing the Response interface.
func NewResponse(status int, body interface{}) Response {
	return HttpResponse{status: status, body: body}
}

var _ Response = (*HttpResponse)(nil)

type HttpResponse struct {
	status int
	body   interface{}
}

func (resp HttpResponse) Status() int       { return resp.status }
func (resp HttpResponse) Body() interface{} { return resp.body }

// HttpError represents any HTTP response indicating an error.
// Use [NewError] to construct a new error.
type HttpError struct {
	status int
	msg    string
}

var _ json.Marshaler = (*HttpError)(nil)
var _ error = (*HttpError)(nil)

func (e HttpError) Error() string { return e.msg }

func (e HttpError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"error": e.msg})
}

// NewError returns a new HttpError with given status and error message.
func NewError(status int, msg string) error {
	return HttpError{
		msg:    msg,
		status: status,
	}
}

// NewGenericError returns a new HttpError with given status and its text as error message.
func NewGenericError(status int) error {
	return HttpError{
		status: status,
		msg:    http.StatusText(status),
	}
}

// respondError takes an httpError and writes it to the given http.ResponseWriter.
// Returns an error if it fails to encode the httpError to JSON or fails
// to write to the http.ResponseWriter.
func respondError(w http.ResponseWriter, err HttpError) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(err.status)
	return json.NewEncoder(w).Encode(err)
}

// respond takes a Response and writes it to the given http.ResponseWriter.
// Returns an error if it fails to encode the response body to JSON or fails
// to write to the http.ResponseWriter.
func respond(w http.ResponseWriter, resp Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status())

	return json.NewEncoder(w).Encode(resp.Body())
}
