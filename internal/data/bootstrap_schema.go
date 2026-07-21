package data

import (
	"context"
	"fmt"

	"github.com/npanel-dev/NPanel-backend/internal/database"
)

func hasExistingLegacySchema(ctx context.Context, driverName, dataSourceName string) (bool, error) {
	// Once a core old-project table exists, startup must not let Ent rewrite the schema.
	for _, tableName := range []string{"user_subscribe", "user", "system"} {
		exists, err := hasDatabaseTable(ctx, driverName, dataSourceName, tableName)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}

func hasDatabaseTable(ctx context.Context, driverName, dataSourceName, tableName string) (bool, error) {
	driver, err := database.NormalizeDriver(driverName)
	if err != nil {
		return false, err
	}
	db, err := database.OpenSQL(driver, dataSourceName)
	if err != nil {
		return false, fmt.Errorf("open sql db: %w", err)
	}
	defer db.Close()

	exists, err := database.TableExists(ctx, db, driver, tableName)
	if err != nil {
		return false, fmt.Errorf("query table %s existence failed: %w", tableName, err)
	}
	return exists, nil
}
