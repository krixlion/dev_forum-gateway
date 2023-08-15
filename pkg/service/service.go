package service

import (
	"context"

	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/otel/trace"
)

type EntityService struct {
	logger logging.Logger
	tracer trace.Tracer
}

type Dependencies struct {
	Logger logging.Logger
	Tracer trace.Tracer
}

func MakeEntityService(d Dependencies) EntityService {
	s := EntityService{
		logger: d.Logger,
		tracer: d.Tracer,
	}

	return s
}

func (s *EntityService) Run(ctx context.Context) {}

func (s *EntityService) Close() error { return nil }
