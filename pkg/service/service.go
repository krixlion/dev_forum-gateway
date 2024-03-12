package service

import (
	"context"

	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/otel/trace"
)

type GatewayService struct {
	logger logging.Logger
	tracer trace.Tracer
}

type Dependencies struct {
	Logger logging.Logger
	Tracer trace.Tracer
}

func MakeGatewayService(d Dependencies) GatewayService {
	s := GatewayService{
		logger: d.Logger,
		tracer: d.Tracer,
	}

	return s
}

func (s *GatewayService) Run(ctx context.Context) {}

func (s *GatewayService) Close() error { return nil }
