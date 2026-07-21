package migrate

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/npanel-dev/NPanel-backend/ent/proxyauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxypayment"
	"github.com/npanel-dev/NPanel-backend/ent/proxyschemamigrations"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribeapplication"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

const currentSchemaVersion int64 = 2156

//go:embed baseline_data.json
var baselineDataJSON []byte

type portableBaseline struct {
	AuthMethods  []baselineAuthMethod  `json:"auth_methods"`
	Payments     []baselinePayment     `json:"payments"`
	Systems      []baselineSystem      `json:"systems"`
	Applications []baselineApplication `json:"applications"`
}

type baselineAuthMethod struct {
	Method  string `json:"method"`
	Config  string `json:"config"`
	Enabled int    `json:"enabled"`
}

type baselinePayment struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Domain      string `json:"domain"`
	Config      string `json:"config"`
	FeeMode     uint   `json:"fee_mode"`
	FeePercent  int64  `json:"fee_percent"`
	FeeAmount   int64  `json:"fee_amount"`
	Sort        int32  `json:"sort"`
	Enable      int    `json:"enable"`
	Token       string `json:"token"`
}

type baselineSystem struct {
	Category    string `json:"category"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type baselineApplication struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Icon              string `json:"icon"`
	Description       string `json:"description"`
	Scheme            string `json:"scheme"`
	UserAgent         string `json:"user_agent"`
	IsDefault         int    `json:"is_default"`
	SubscribeTemplate string `json:"subscribe_template"`
	OutputFormat      string `json:"output_format"`
	DownloadLink      string `json:"download_link"`
}

func loadPortableBaseline() (*portableBaseline, error) {
	var baseline portableBaseline
	if err := json.Unmarshal(baselineDataJSON, &baseline); err != nil {
		return nil, fmt.Errorf("decode portable baseline: %w", err)
	}
	return &baseline, nil
}

func (m *Migrator) InitPortableBaseline(ctx context.Context) error {
	baseline, err := loadPortableBaseline()
	if err != nil {
		return err
	}

	for _, seed := range baseline.AuthMethods {
		exists, err := m.client.ProxyAuthMethod.Query().Where(proxyauthmethod.MethodEQ(seed.Method)).Exist(ctx)
		if err != nil {
			return fmt.Errorf("query auth method %s: %w", seed.Method, err)
		}
		if exists {
			continue
		}
		if _, err := m.client.ProxyAuthMethod.Create().
			SetMethod(seed.Method).
			SetConfig(seed.Config).
			SetEnabled(seed.Enabled != 0).
			Save(ctx); err != nil {
			return fmt.Errorf("create auth method %s: %w", seed.Method, err)
		}
	}

	for _, seed := range baseline.Payments {
		exists, err := m.client.ProxyPayment.Query().Where(proxypayment.PlatformEQ(seed.Platform)).Exist(ctx)
		if err != nil {
			return fmt.Errorf("query payment %s: %w", seed.Platform, err)
		}
		if exists {
			continue
		}
		if _, err := m.client.ProxyPayment.Create().
			SetID(seed.ID).
			SetName(seed.Name).
			SetPlatform(seed.Platform).
			SetDescription(seed.Description).
			SetIcon(seed.Icon).
			SetDomain(seed.Domain).
			SetConfig(seed.Config).
			SetFeeMode(seed.FeeMode).
			SetFeePercent(seed.FeePercent).
			SetFeeAmount(seed.FeeAmount).
			SetSort(seed.Sort).
			SetEnable(seed.Enable != 0).
			SetToken(seed.Token).
			Save(ctx); err != nil {
			return fmt.Errorf("create payment %s: %w", seed.Platform, err)
		}
	}

	for _, seed := range baseline.Systems {
		exists, err := m.client.ProxySystem.Query().Where(proxysystem.KeyEQ(seed.Key)).Exist(ctx)
		if err != nil {
			return fmt.Errorf("query system setting %s: %w", seed.Key, err)
		}
		if exists {
			continue
		}
		if _, err := m.client.ProxySystem.Create().
			SetCategory(seed.Category).
			SetKey(seed.Key).
			SetValue(seed.Value).
			SetType(seed.Type).
			SetDesc(seed.Description).
			Save(ctx); err != nil {
			return fmt.Errorf("create system setting %s: %w", seed.Key, err)
		}
	}

	for _, seed := range baseline.Applications {
		exists, err := m.client.ProxySubscribeApplication.Query().
			Where(proxysubscribeapplication.NameEQ(seed.Name)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("query subscribe application %s: %w", seed.Name, err)
		}
		if exists {
			continue
		}
		idUsed, err := m.client.ProxySubscribeApplication.Query().
			Where(proxysubscribeapplication.IDEQ(seed.ID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("query subscribe application id %d: %w", seed.ID, err)
		}
		if idUsed {
			return fmt.Errorf("cannot seed subscribe application %s: id %d is already in use", seed.Name, seed.ID)
		}
		if _, err := m.client.ProxySubscribeApplication.Create().
			SetID(seed.ID).
			SetName(seed.Name).
			SetIcon(seed.Icon).
			SetDescription(seed.Description).
			SetScheme(seed.Scheme).
			SetUserAgent(seed.UserAgent).
			SetIsDefault(seed.IsDefault != 0).
			SetSubscribeTemplate(seed.SubscribeTemplate).
			SetOutputFormat(seed.OutputFormat).
			SetDownloadLink(seed.DownloadLink).
			Save(ctx); err != nil {
			return fmt.Errorf("create subscribe application %s: %w", seed.Name, err)
		}
	}

	if err := m.syncPostgresApplicationSequence(ctx); err != nil {
		return err
	}
	if err := m.markPortableBaseline(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) syncPostgresApplicationSequence(ctx context.Context) error {
	driver, err := database.NormalizeDriver(m.dbDriver)
	if err != nil {
		return err
	}
	if driver != database.Postgres {
		return nil
	}
	db, err := database.OpenSQL(driver, m.dbSource)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('subscribe_application', 'id'), COALESCE((SELECT MAX(id) FROM subscribe_application), 1), true)`); err != nil {
		return fmt.Errorf("sync PostgreSQL subscribe_application sequence: %w", err)
	}
	return nil
}

func (m *Migrator) markPortableBaseline(ctx context.Context) error {
	exists, err := m.client.ProxySchemaMigrations.Query().
		Where(proxyschemamigrations.IDEQ(currentSchemaVersion)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("query portable baseline version: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := m.client.ProxySchemaMigrations.Create().
		SetID(currentSchemaVersion).
		SetDirty(false).
		Save(ctx); err != nil {
		return fmt.Errorf("record portable baseline version %d: %w", currentSchemaVersion, err)
	}
	return nil
}
