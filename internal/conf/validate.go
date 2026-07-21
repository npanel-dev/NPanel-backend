package conf

import (
	"fmt"
	"strings"

	"github.com/npanel-dev/NPanel-backend/internal/database"
)

// ValidateBootstrap checks the configuration required before dependency
// injection starts. This keeps malformed or incomplete config files from
// surfacing later as nil-pointer panics.
func ValidateBootstrap(c *Bootstrap) error {
	if c == nil {
		return fmt.Errorf("bootstrap config is required")
	}
	if c.Server == nil {
		return fmt.Errorf("server config is required")
	}
	if c.Server.Http == nil {
		return fmt.Errorf("server.http config is required")
	}
	if strings.TrimSpace(c.Server.Http.Addr) == "" {
		return fmt.Errorf("server.http.addr is required")
	}
	if c.Server.Grpc == nil {
		return fmt.Errorf("server.grpc config is required")
	}
	if strings.TrimSpace(c.Server.Grpc.Addr) == "" {
		return fmt.Errorf("server.grpc.addr is required")
	}
	if c.App == nil {
		return fmt.Errorf("app config is required")
	}
	return ValidateData(c.Data)
}

// ValidateData checks database and Redis settings used directly by NewData.
func ValidateData(c *Data) error {
	if c == nil {
		return fmt.Errorf("data config is required")
	}
	if c.Database == nil {
		return fmt.Errorf("data.database config is required")
	}
	if _, err := database.NormalizeDriver(c.Database.Driver); err != nil {
		return fmt.Errorf("data.database.driver: %w", err)
	}
	if strings.TrimSpace(c.Database.Source) == "" {
		return fmt.Errorf("data.database.source is required")
	}
	if c.Redis == nil {
		return fmt.Errorf("data.redis config is required")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("data.redis.addr is required")
	}
	if c.Redis.ReadTimeout == nil {
		return fmt.Errorf("data.redis.read_timeout is required")
	}
	if c.Redis.WriteTimeout == nil {
		return fmt.Errorf("data.redis.write_timeout is required")
	}
	return nil
}
