package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	authpb "github.com/krixlion/dev_forum-auth/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-auth/pkg/tokens/translator"
	"github.com/krixlion/dev_forum-gateway/pkg/api"
	"github.com/krixlion/dev_forum-gateway/pkg/service"
	"github.com/krixlion/dev_forum-lib/cert"
	"github.com/krixlion/dev_forum-lib/env"
	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	articlepb "github.com/krixlion/dev_forum-article/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-lib/tracing"
	userpb "github.com/krixlion/dev_forum-user/pkg/grpc/v1"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var port int
var isTLS bool

func init() {
	portFlag := flag.Int("p", 80, "The HTTP server port.")
	insecureFlag := flag.Bool("insecure", false, "Whether to use TLS.")
	flag.Parse()
	port = *portFlag
	isTLS = !(*insecureFlag)
}

// Hardcoded root dir name.
const projectDir = "app"
const serviceName = "gateway"

func main() {
	env.Load(projectDir)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	deps, err := getServiceDependencies(ctx, serviceName, port, isTLS)
	if err != nil {
		logging.Log("Failed to initialize service dependencies", "err", err)
		return
	}

	service := service.MakeGatewayService(deps)

	go func() {
		if err := service.Run(); err != nil {
			logging.Log("Failed to run service", "err", err)
		}
	}()

	<-ctx.Done()
	logging.Log("Service shutting down")

	defer func() {
		cancel()

		if err := service.Close(); err != nil {
			logging.Log("Failed to shutdown service", "err", err)
			return
		}

		logging.Log("Service shutdown successful")
	}()
}

// getServiceDependencies is a Composition root.
// Panics on any non-nil error.
func getServiceDependencies(ctx context.Context, serviceName string, port int, isTLS bool) (service.Dependencies, error) {
	shutdownTracing, err := tracing.InitProvider(ctx, serviceName, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		return service.Dependencies{}, err
	}

	tracer := otel.Tracer(serviceName)

	logger, err := logging.NewLogger()
	if err != nil {
		return service.Dependencies{}, err
	}

	var creds = insecure.NewCredentials()
	if isTLS {
		caCertPool, err := cert.LoadCaPool(os.Getenv("TLS_CA_PATH"))
		if err != nil {
			return service.Dependencies{}, err
		}

		serverCert, err := cert.LoadX509KeyPair(os.Getenv("TLS_CERT_PATH"), os.Getenv("TLS_KEY_PATH"))
		if err != nil {
			return service.Dependencies{}, err
		}

		clientCert, err := cert.LoadX509KeyPair(os.Getenv("TLS_CLIENT_CERT_PATH"), os.Getenv("TLS_CLIENT_KEY_PATH"))
		if err != nil {
			return service.Dependencies{}, err
		}

		tlsConfig := &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{clientCert, serverCert},
		}

		creds = credentials.NewTLS(tlsConfig)
	}

	authConn, err := grpc.NewClient(os.Getenv("AUTH_SERVICE_SERVICE_HOST")+":"+os.Getenv("AUTH_SERVICE_SERVICE_PORT"),
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return service.Dependencies{}, err
	}
	authClient := authpb.NewAuthServiceClient(authConn)

	userConn, err := grpc.NewClient(os.Getenv("USER_SERVICE_SERVICE_HOST")+":"+os.Getenv("USER_SERVICE_SERVICE_PORT"),
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return service.Dependencies{}, err
	}

	articleConn, err := grpc.NewClient(os.Getenv("ARTICLE_SERVICE_SERVICE_HOST")+":"+os.Getenv("ARTICLE_SERVICE_SERVICE_PORT"),
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return service.Dependencies{}, err
	}

	translatorConfig := translator.Config{
		StreamRenewalInterval: time.Second * 10,
		JobQueueSize:          1,
	}
	translator := translator.NewTranslator(authClient, translatorConfig, translator.WithLogger(logger))
	go translator.Run(ctx)

	router := chi.NewRouter()
	router.Mount("/articles", api.MakeArticleHandler(articlepb.NewArticleServiceClient(articleConn), translator, logger))
	router.Mount("/users", api.MakeUserHandler(userpb.NewUserServiceClient(userConn), translator, logger))
	router.Mount("/auth", api.MakeAuthHandler(authClient, logger))

	httpServer := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	return service.Dependencies{
		Logger:     logger,
		Tracer:     tracer,
		HttpServer: httpServer,
		ShutdownFunc: func() error {
			return errors.Join(httpServer.Shutdown(ctx), userConn.Close(), articleConn.Close(), shutdownTracing(), logger.Sync())
		},
	}, nil
}
