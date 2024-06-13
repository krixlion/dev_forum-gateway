package service

import (
	"context"
	"net/http"

	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/otel/trace"
)

type GatewayService struct {
	logger     logging.Logger
	tracer     trace.Tracer
	HttpServer *http.Server
}

type Dependencies struct {
	Logger     logging.Logger
	Tracer     trace.Tracer
	HttpServer *http.Server
}

func MakeGatewayService(d Dependencies) GatewayService {
	s := GatewayService{
		logger:     d.Logger,
		tracer:     d.Tracer,
		HttpServer: d.HttpServer,
	}

	return s
}

func (s *GatewayService) Run() error {
	return s.HttpServer.ListenAndServe()
}

func (s *GatewayService) Shutdown(ctx context.Context) error {
	return s.HttpServer.Shutdown(ctx)
}
