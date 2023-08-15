package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	rabbitmq "github.com/krixlion/dev-forum_rabbitmq"
	pb "github.com/krixlion/dev_forum-entity/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-lib/env"
	"github.com/krixlion/dev_forum-lib/event"
	"github.com/krixlion/dev_forum-lib/event/broker"
	"github.com/krixlion/dev_forum-lib/event/dispatcher"
	"github.com/krixlion/dev_forum-lib/logging"
	"github.com/krixlion/dev_forum-lib/tracing"
	"github.com/krixlion/dev_forum-user/pkg/storage"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var port int

func init() {
	portFlag := flag.Int("p", 50051, "The gRPC server port")
	flag.Parse()
	port = *portFlag
}

// Hardcoded root dir name.
const projectDir = "app"
const serviceName = "entity-service"

func main() {
	env.Load(projectDir)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	shutdownTracing, err := tracing.InitProvider(ctx, serviceName)
	if err != nil {
		logging.Log("Failed to initialize tracing", "err", err)
	}

	service := service.NewEntityService(port, getServiceDependencies())
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

	cmdPort := os.Getenv("DB_WRITE_PORT")
	cmdHost := os.Getenv("DB_WRITE_HOST")
	cmdUser := os.Getenv("DB_WRITE_USER")
	cmdPass := os.Getenv("DB_WRITE_PASS")
	// cmd, err := eventstore.MakeDB(cmdPort, cmdHost, cmdUser, cmdPass, logger, tracer)
	if err != nil {
		panic(err)
	}

	queryPort := os.Getenv("DB_READ_PORT")
	queryHost := os.Getenv("DB_READ_HOST")
	queryPass := os.Getenv("DB_READ_PASS")
	// query, err := query.MakeDB(queryHost, queryPort, queryPass, logger, tracer)
	if err != nil {
		panic(err)
	}

	storage := storage.NewCQRStorage(cmd, query, logger, tracer)

	mqPort := os.Getenv("MQ_PORT")
	mqHost := os.Getenv("MQ_HOST")
	mqUser := os.Getenv("MQ_USER")
	mqPass := os.Getenv("MQ_PASS")
	consumer := serviceName
	mqConfig := rabbitmq.Config{
		QueueSize:         100,
		MaxWorkers:        100,
		ReconnectInterval: time.Second * 2,
		MaxRequests:       30,
		ClearInterval:     time.Second * 5,
		ClosedTimeout:     time.Second * 15,
	}

	messageQueue := rabbitmq.NewRabbitMQ(
		consumer,
		mqUser,
		mqPass,
		mqHost,
		mqPort,
		mqConfig,
		rabbitmq.WithLogger(logger),
		rabbitmq.WithTracer(tracer),
	)

	broker := broker.NewBroker(messageQueue, logger, tracer)
	dispatcher := dispatcher.NewDispatcher(broker, 20)
	dispatcher.SetSyncHandler(event.HandlerFunc(storage.CatchUp))

	for eType, handlers := range storage.EventHandlers() {
		dispatcher.Subscribe(eType, handlers...)
	}

	entityServer := server.NewEntityServer(storage, logger, dispatcher)

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),

		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zap.L()),
			otelgrpc.UnaryServerInterceptor(),
			entityServer.ValidateRequestInterceptor(),
		),
	)

	reflection.Register(grpcServer)
	pb.RegisterEntityServiceServer(grpcServer, entityServer)

	return service.Dependencies{
		Logger:     logger,
		Broker:     broker,
		GRPCServer: grpcServer,
		SyncEvents: &cmd,
		Storage:    storage,
		Dispatcher: dispatcher,
		ShutdownFunc: func() error {
			grpcServer.GracefulStop()
			return entityServer.Close()
		},
	}
}
