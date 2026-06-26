package main

import (
	"flag"
	"fmt"
	"os"

	bootstraplog "github.com/npanel-dev/NPanel-backend/internal/bootstrap/logging"
	"github.com/npanel-dev/NPanel-backend/internal/buildmeta"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	npanelLogger "github.com/npanel-dev/NPanel-backend/pkg/logger"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software
	Name string
	// Version is the version of the compiled software
	Version string
	// flagconf is the config flag
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()

	if Name == "" {
		Name = "npanel"
	}
	if Version == "" {
		Version = "v1.0.10"
	}
	buildmeta.SetMainVersion(Version)

	// 抑制 Redis 客户端的警告信息
	os.Setenv("REDIS_LOG_LEVEL", "ERROR")

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	conf.SetLegacyDebugMode(bc.GetDebug())

	logConfig := bootstraplog.DefaultConfig(Name)
	if value := c.Value("log"); value.Load() != nil {
		if err := value.Scan(&logConfig); err != nil {
			panic(fmt.Errorf("scan log config: %w", err))
		}
	}

	zapLogger, closeLogger, err := bootstraplog.New(logConfig, id, Name, Version)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := closeLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "close logger: %v\n", err)
		}
	}()

	npanelLogger.SetWriter(bootstraplog.NewNPanelWriter(zapLogger))

	logger := log.With(
		bootstraplog.NewKratosLogger(zapLogger),
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.App, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
