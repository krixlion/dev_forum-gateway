package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/krixlion/dev_forum-gateway/pkg/service"
	"github.com/krixlion/dev_forum-lib/env"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	"go.opentelemetry.io/otel"
)

// var port int

// func init() {
// 	portFlag := flag.Int("p", 80, "The GraphQL server port")
// 	flag.Parse()
// 	port = *portFlag
// }

// Hardcoded root dir name.
const projectDir = "app"
const serviceName = "gateway"

func main() {
	env.Load(projectDir)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	shutdownTracing, err := tracing.InitProvider(ctx, serviceName)
	if err != nil {
		logging.Log("Failed to initialize tracing", "err", err)
	}

	service := service.MakeEntityService(getServiceDependencies())
	service.Run(ctx)

	<-ctx.Done()
	logging.Log("Service shutting down")

	defer func() {
		cancel()
		shutdownTracing()
		err := service.Close()
		if err != nil {
			logging.Log("Failed to shutdown service", "err", err)
		} else {
			logging.Log("Service shutdown properly")
		}
	}()
}

// getServiceDependencies is a Composition root.
// Panics on any non-nil error.
func getServiceDependencies() service.Dependencies {
	tracer := otel.Tracer(serviceName)

	logger, err := logging.NewLogger()
	if err != nil {
		panic(err)
	}

	return service.Dependencies{
		Logger: logger,
		Tracer: tracer,
	}
}
