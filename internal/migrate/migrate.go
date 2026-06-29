package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	entsqlschema "entgo.io/ent/dialect/sql/schema"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyauthmethod"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/model/auth"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"

	"github.com/go-kratos/kratos/v2/log"
)

// Migrator 数据迁移器
type Migrator struct {
	client   *ent.Client
	logger   *log.Helper
	appConf  *conf.Application
	dbDriver string
	dbSource string
}

// NewMigrator 创建新的迁移器
func NewMigrator(client *ent.Client, logger log.Logger, appConf *conf.Application, dbDriver, dbSource string) *Migrator {
	return &Migrator{
		client:   client,
		logger:   log.NewHelper(logger),
		appConf:  appConf,
		dbDriver: dbDriver,
		dbSource: dbSource,
	}
}

// AutoMigrate 自动迁移数据库结构
func (m *Migrator) AutoMigrate(ctx context.Context) error {
	m.logger.Info("Starting auto migration...")

	if err := m.client.Schema.Create(ctx); err != nil {
		m.logger.Errorf("Failed to create schema: %v", err)
		return fmt.Errorf("failed to create schema: %w", err)
	}

	m.logger.Info("Auto migration completed successfully")
	return nil
}

// AutoMigrateLegacySchema brings imported legacy databases up to the current
// Ent schema without deleting legacy-only columns or indexes.
func (m *Migrator) AutoMigrateLegacySchema(ctx context.Context) error {
	m.logger.Info("Starting legacy schema compatibility migration...")

	if err := m.client.Schema.Create(
		ctx,
		entsqlschema.WithDropColumn(false),
		entsqlschema.WithDropIndex(false),
		entsqlschema.WithForeignKeys(false),
	); err != nil {
		m.logger.Errorf("Failed to create legacy compatible schema: %v", err)
		return fmt.Errorf("failed to create legacy compatible schema: %w", err)
	}

	m.logger.Info("Legacy schema compatibility migration completed successfully")
	return nil
}

// AutoMigrateWithData 自动迁移数据库结构并初始化数据
func (m *Migrator) AutoMigrateWithData(ctx context.Context) error {
	m.logger.Info("Starting auto migration with data initialization...")

	// 先执行数据库结构迁移
	if err := m.AutoMigrate(ctx); err != nil {
		return err
	}

	// 初始化基础数据
	if err := m.InitBasicData(ctx); err != nil {
		return fmt.Errorf("failed to initialize basic data: %w", err)
	}

	// 创建默认管理员用户
	if err := m.CreateDefaultAdminUser(ctx); err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	m.logger.Info("Auto migration with data completed successfully")
	return nil
}

// InitBasicData 初始化基础数据
func (m *Migrator) InitBasicData(ctx context.Context) error {
	m.logger.Info("Starting basic data initialization...")
	if err := m.initLegacyDefaultData(ctx); err != nil {
		return fmt.Errorf("failed to init legacy default data: %w", err)
	}
	if err := m.ensureEmailAuthMethodTemplates(ctx); err != nil {
		return fmt.Errorf("failed to ensure email auth method templates: %w", err)
	}

	m.logger.Info("Basic data initialization completed")
	return nil
}

// CreateDefaultAdminUser 创建默认管理员用户
func (m *Migrator) CreateDefaultAdminUser(ctx context.Context) error {
	// 从配置文件读取管理员凭据
	var email, password string
	if m.appConf != nil && m.appConf.Admin != nil {
		email = m.appConf.Admin.Email
		password = m.appConf.Admin.Password
	}

	// 如果配置为空，使用默认值
	if email == "" {
		email = "admin@example.com"
	}
	if password == "" {
		password = "admin123456"
	}

	// 旧项目是“只要库里已有用户，就跳过管理员初始化”
	exist, err := m.client.ProxyUser.Query().
		Exist(ctx)
	if err != nil {
		return err
	}

	if exist {
		m.logger.Infof("User already exists, skip creating administrator account")
		return nil
	}

	encodedPwd := tool.EncodePassWord(password)
	referCode := uuidx.UserInviteCode(time.Now().Unix())

	// 创建管理员用户
	user, err := m.client.ProxyUser.Create().
		SetPassword(encodedPwd).
		SetReferCode(referCode).
		SetIsAdmin(true).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// 创建邮箱认证方式
	_, err = m.client.ProxyUserAuthMethod.Create().
		SetUserID(user.ID).
		SetAuthType("email").
		SetAuthIdentifier(email).
		SetVerified(true).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create admin auth method: %w", err)
	}

	m.logger.Infof("Default admin user created successfully with email: %s", email)
	m.logger.Infof("Default admin credentials - Email: %s, Password: %s", email, password)
	m.logger.Infof("Please change the default password immediately after first login!")
	return nil
}

func (m *Migrator) ensureEmailAuthMethodTemplates(ctx context.Context) error {
	method, err := m.client.ProxyAuthMethod.Query().
		Where(proxyauthmethod.Method("email")).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to query email auth method: %w", err)
	}

	var raw auth.EmailAuthConfig
	if err := json.Unmarshal([]byte(method.Config), &raw); err != nil {
		return fmt.Errorf("failed to parse email auth config: %w", err)
	}

	needsUpdate := raw.VerifyEmailTemplate == "" ||
		raw.ExpirationEmailTemplate == "" ||
		raw.MaintenanceEmailTemplate == "" ||
		raw.TrafficExceedEmailTemplate == ""
	if !needsUpdate {
		return nil
	}

	var config auth.EmailAuthConfig
	config.Unmarshal(method.Config)

	_, err = m.client.ProxyAuthMethod.UpdateOneID(method.ID).
		SetConfig(config.Marshal()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update email auth method templates: %w", err)
	}

	m.logger.Info("Email auth method templates backfilled with defaults")
	return nil
}
