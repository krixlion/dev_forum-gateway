package service

import (
	"net/http"

	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/otel/trace"
)

type GatewayService struct {
	logger     logging.Logger
	tracer     trace.Tracer
	httpServer *http.Server
	shutdown   func() error
}

type Dependencies struct {
	Logger       logging.Logger
	Tracer       trace.Tracer
	HttpServer   *http.Server
	ShutdownFunc func() error
}

func MakeGatewayService(d Dependencies) GatewayService {
	s := GatewayService{
		logger:     d.Logger,
		tracer:     d.Tracer,
		httpServer: d.HttpServer,
	}

	return s
}

func (s *GatewayService) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *GatewayService) Close() error {
	return s.shutdown()
}
