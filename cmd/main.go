package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/krixlion/dev_forum-gateway/pkg/api"
	"github.com/krixlion/dev_forum-gateway/pkg/service"
	"github.com/krixlion/dev_forum-lib/env"
	"github.com/krixlion/dev_forum-lib/logging"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

var port int
var isTLS bool

func init() {
	portFlag := flag.Int("p", 80, "The GraphQL server port.")
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

	deps, err := getServiceDependencies(ctx, serviceName, isTLS)
	if err != nil {
		logging.Log("Failed to initialize service dependencies", "err", err)
		return
	}

	service := service.MakeGatewayService(port, deps)

	go func() {
		if err := service.Run(); err != nil {
			logging.Log("Failed to run service", "err", err)
		}
	}()

	<-ctx.Done()
	logging.Log("Service shutting down")

	defer func() {
		cancel()

		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer closeCancel()

		if err := service.Shutdown(closeCtx); err != nil {
			logging.Log("Failed to shutdown service", "err", err)
			return
		}

		logging.Log("Service shutdown successful")
	}()
}

// getServiceDependencies is a Composition root.
// Panics on any non-nil error.
func getServiceDependencies(ctx context.Context, serviceName string, isTLS bool) (service.Dependencies, error) {
	tracer := otel.Tracer(serviceName)

	logger, err := logging.NewLogger()
	if err != nil {
		return service.Dependencies{}, err
	}

	userConn, err := grpc.DialContext(ctx, "")
	if err != nil {
		return service.Dependencies{}, err
	}

	articleConn, err := grpc.DialContext(ctx, "")
	if err != nil {
		return service.Dependencies{}, err
	}

	router := chi.NewRouter()
	router.Mount("/articles", api.MakeArticleHandler(articleConn))
	router.Mount("/users", api.MakeUserHandler(userConn))

	var tlsConfig *tls.Config
	if isTLS {
		cert, err := tls.LoadX509KeyPair("server.cert", "server.key")
		if err != nil {
			return service.Dependencies{}, err
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	httpServer := &http.Server{
		Handler:   router,
		TLSConfig: tlsConfig,
		Addr:      "0.0.0.0:" + os.Getenv("PORT"),
	}

	return service.Dependencies{
		Logger:     logger,
		Tracer:     tracer,
		HttpServer: httpServer,
	}, nil
}
