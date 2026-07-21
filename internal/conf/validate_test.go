package conf

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"google.golang.org/protobuf/types/known/durationpb"
)

func validBootstrapConfig(driver string) *Bootstrap {
	return &Bootstrap{
		Server: &Server{
			Http: &Server_HTTP{Addr: "127.0.0.1:8081"},
			Grpc: &Server_GRPC{Addr: "127.0.0.1:9012"},
		},
		Data: &Data{
			Database: &Data_Database{Driver: driver, Source: "test-source"},
			Redis: &Data_Redis{
				Addr:         "127.0.0.1:6379",
				ReadTimeout:  durationpb.New(200000000),
				WriteTimeout: durationpb.New(200000000),
			},
		},
		App: &Application{},
	}
}

func TestValidateBootstrapAcceptsSupportedDatabaseDrivers(t *testing.T) {
	for _, driver := range []string{"mysql", "postgres", "postgresql"} {
		t.Run(driver, func(t *testing.T) {
			if err := ValidateBootstrap(validBootstrapConfig(driver)); err != nil {
				t.Fatalf("ValidateBootstrap(%q): %v", driver, err)
			}
		})
	}
}

func TestValidateBootstrapRejectsIncompleteConfig(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Bootstrap)
		want string
	}{
		{"missing server", func(c *Bootstrap) { c.Server = nil }, "server config"},
		{"missing http", func(c *Bootstrap) { c.Server.Http = nil }, "server.http"},
		{"missing http address", func(c *Bootstrap) { c.Server.Http.Addr = " " }, "server.http.addr"},
		{"missing grpc", func(c *Bootstrap) { c.Server.Grpc = nil }, "server.grpc"},
		{"missing grpc address", func(c *Bootstrap) { c.Server.Grpc.Addr = " " }, "server.grpc.addr"},
		{"missing app", func(c *Bootstrap) { c.App = nil }, "app config"},
		{"missing data", func(c *Bootstrap) { c.Data = nil }, "data config"},
		{"missing database", func(c *Bootstrap) { c.Data.Database = nil }, "data.database"},
		{"unsupported driver", func(c *Bootstrap) { c.Data.Database.Driver = "sqlite" }, "unsupported database driver"},
		{"missing source", func(c *Bootstrap) { c.Data.Database.Source = " " }, "data.database.source"},
		{"missing redis", func(c *Bootstrap) { c.Data.Redis = nil }, "data.redis"},
		{"missing redis address", func(c *Bootstrap) { c.Data.Redis.Addr = " " }, "data.redis.addr"},
		{"missing redis timeout", func(c *Bootstrap) { c.Data.Redis.ReadTimeout = nil }, "read_timeout"},
		{"missing redis write timeout", func(c *Bootstrap) { c.Data.Redis.WriteTimeout = nil }, "write_timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validBootstrapConfig("mysql")
			tt.edit(config)
			err := ValidateBootstrap(config)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateBootstrap() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestExampleConfigsLoadAndValidate(t *testing.T) {
	for _, name := range []string{"config.docker.yaml", "config.postgres.yaml"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "configs", name)
			loader := config.New(config.WithSource(file.NewSource(path)))
			t.Cleanup(func() { _ = loader.Close() })
			if err := loader.Load(); err != nil {
				t.Fatalf("load %s: %v", path, err)
			}
			var bootstrap Bootstrap
			if err := loader.Scan(&bootstrap); err != nil {
				t.Fatalf("scan %s: %v", path, err)
			}
			if err := ValidateBootstrap(&bootstrap); err != nil {
				t.Fatalf("validate %s: %v", path, err)
			}
		})
	}
}
