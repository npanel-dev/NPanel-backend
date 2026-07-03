package data

import (
	"context"
	"fmt"
	"sync"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/migrate"
	"github.com/npanel-dev/NPanel-backend/internal/queue/handler"
	"github.com/npanel-dev/NPanel-backend/internal/service"
	"github.com/npanel-dev/NPanel-backend/pkg/device"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

// Global device manager instance
var (
	globalDeviceManager *device.DeviceManager
	deviceManagerOnce   sync.Once
)

// ProviderSet is data providers
var ProviderSet = wire.NewSet(
	NewData,
	NewEntClient,
	NewDeviceManager,
	NewAdsRepo,
	NewAdminAnnouncementRepo,
	NewAdminAuthMethodRepo,
	NewAdminConsoleRepo,
	NewCouponRepo,
	NewAdminDocumentRepo,
	NewAdminSystemLogRepo,
	NewAdminTrafficLogRepo,
	NewAdminLogSettingRepo,
	NewAdminMarketingRepo,
	NewOrderRepo,
	NewAdminPaymentRepo,
	NewAdminRoutingRepo,
	NewAdminServerRepo,
	NewAdminNodeRepo,
	NewAdminMigrationRepo,
	NewSubscribeApplicationRepo,
	NewSubscribeRepo,
	NewAdminSystemRepo,
	NewTicketRepo,
	NewAdminRedemptionRepo,
	NewAdminGroupRepo,
	NewAdminWithdrawalRepo,
	// Admin User模块仓储
	NewAdminUserRepo,
	NewAdminUserAuthMethodRepo,
	NewAdminUserDeviceRepo,
	NewAdminUserSubscribeRepo,
	// Auth模块仓储
	NewAuthRepo,
	NewAuthCompat,
	// Public Common模块仓储
	NewCommonRepo,
	// Public Order模块仓储
	NewPublicOrderRepo,
	// Public Announcement模块仓储
	NewPublicAnnouncementRepo,
	// Public Document模块仓储
	NewPublicDocumentRepo,
	// Public Portal模块仓储
	NewPublicPortalRepo,
	// Public Redemption模块仓储
	NewPublicRedemptionRepo,
	// Public Ticket模块仓储
	NewPublicTicketRepo,
	// Public User模块仓储
	NewPublicUserRepo,
	// Public Payment模块仓储
	NewPublicPaymentRepo,
	// Public Subscribe模块仓储
	NewPublicSubscribeRepo,
	// Public Subscription模块仓储（订阅配置生成）
	NewPublicSubscriptionRepo,
	// Public Withdrawal模块仓储
	NewWithdrawalRepo,
	// Server模块仓储
	NewServerNodeRepo,
	// Auth OAuth模块仓储
	NewOAuthRepo,
)

// Data
type Data struct {
	db             *ent.Client
	rdb            *redis.Client
	queue          *asynq.Client
	queueServer    *asynq.Server
	queueScheduler *asynq.Scheduler
	serverConf     *conf.Server
	conf           *conf.Application     // 应用配置（包含JWT等配置）
	deviceMgr      *device.DeviceManager // 设备管理器
}

// DB 获取数据库客户端
func (d *Data) DB() *ent.Client {
	return d.db
}

// RDB 获取Redis客户端
func (d *Data) RDB() *redis.Client {
	return d.rdb
}

// Queue 获取异步队列客户端
func (d *Data) Queue() *asynq.Client {
	return d.queue
}

// AppConf 获取应用配置
func (d *Data) AppConf() *conf.Application {
	return d.conf
}

// Redis 获取Redis客户端 (为中间件兼容)
func (d *Data) Redis() *redis.Client {
	return d.rdb
}

// DeviceManager 获取设备管理器
func (d *Data) DeviceManager() *device.DeviceManager {
	return d.deviceMgr
}

// FindOne 实现用户服务接口 - 根据用户ID查找用户
func (d *Data) FindOne(ctx context.Context, userId int64) (*ent.ProxyUser, error) {
	return d.db.ProxyUser.Get(ctx, userId)
}

// NewData
func NewData(c *conf.Data, serverConf *conf.Server, appConf *conf.Application, logger log.Logger) (*Data, func(), error) {
	bootstrapCtx := context.Background()
	log.NewHelper(logger).Infof("connecting to database: %s", c.Database.Source)

	client, err := ent.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		log.NewHelper(logger).Errorf("failed opening connection to database: %v", err)
		return nil, nil, err
	}
	client = client.Debug()

	migrator := migrate.NewMigrator(client, logger, appConf, c.Database.Driver, c.Database.Source)
	existingLegacySchema, err := hasExistingLegacySchema(bootstrapCtx, c.Database.Driver, c.Database.Source)
	if err != nil {
		log.NewHelper(logger).Errorf("failed detecting database schema state: %v", err)
		return nil, nil, err
	}

	if existingLegacySchema {
		log.NewHelper(logger).Info("existing legacy schema detected; running safe ent schema compatibility migration")
		hasMigrationsTable, err := hasDatabaseTable(bootstrapCtx, c.Database.Driver, c.Database.Source, "schema_migrations")
		if err != nil {
			log.NewHelper(logger).Errorf("failed checking schema_migrations table: %v", err)
			return nil, nil, err
		}
		if err := migrator.AutoMigrateLegacySchema(bootstrapCtx); err != nil {
			log.NewHelper(logger).Errorf("failed to migrate legacy compatible schema: %v", err)
			return nil, nil, fmt.Errorf("failed to migrate legacy compatible schema: %w", err)
		}
		if hasMigrationsTable {
			if err := migrator.InitBasicData(bootstrapCtx); err != nil {
				log.NewHelper(logger).Errorf("failed to initialize legacy default data: %v", err)
				return nil, nil, fmt.Errorf("failed to initialize legacy default data: %w", err)
			}
		} else {
			log.NewHelper(logger).Warn("schema_migrations table not found; skipping legacy default data sync for existing database")
			if err := migrator.EnsureLegacyCompatibilitySchema(bootstrapCtx); err != nil {
				log.NewHelper(logger).Errorf("failed to ensure legacy compatibility schema: %v", err)
				return nil, nil, fmt.Errorf("failed to ensure legacy compatibility schema: %w", err)
			}
		}
		if err := migrator.CreateDefaultAdminUser(bootstrapCtx); err != nil {
			log.NewHelper(logger).Errorf("failed to ensure default admin user: %v", err)
			return nil, nil, fmt.Errorf("failed to ensure default admin user: %w", err)
		}
	} else {
		log.NewHelper(logger).Info("no existing legacy schema detected; running bootstrap migration path")
		if err := migrator.AutoMigrateWithData(bootstrapCtx); err != nil {
			log.NewHelper(logger).Errorf("failed to initialize schema/data: %v", err)
			return nil, nil, fmt.Errorf("failed to initialize schema/data: %w", err)
		}
	}

	// The old project hydrates its runtime config snapshot from database/auth
	// records during startup. Keep the Kratos runtime config aligned so public
	// routes and middleware observe the same values.
	syncRuntimeAppConfig(bootstrapCtx, client, appConf, log.NewHelper(log.With(logger, "module", "data/runtime_config")))

	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		PoolSize:     int(c.Redis.PoolSize),
		MinIdleConns: int(c.Redis.MinIdleConns),
		// Many customer environments still run Redis variants/versions without
		// CLIENT MAINT_NOTIFICATIONS support. Disable the handshake explicitly
		// to avoid noisy fallback warnings during startup.
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	// 测试 Redis 连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.NewHelper(logger).Errorf("failed connecting to redis: %v", err)
		return nil, nil, err
	}
	log.NewHelper(logger).Infof("connected to redis: %s", c.Redis.Addr)

	// 创建 asynq 客户端 - 用于发送任务到队列
	redisOpt := compatibleRedisConnOpt{
		RedisClientOpt: asynq.RedisClientOpt{
			Addr:     c.Redis.Addr,
			Password: c.Redis.Password,
			DB:       int(c.Redis.Db),
		},
	}
	queueClient := asynq.NewClient(redisOpt)

	// 创建 asynq 服务器 - 用于处理队列中的任务
	queueServer := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10, // 并发处理10个任务
			// Logger 使用默认的 asynq logger
		},
	)
	queueScheduler := newQueueScheduler(redisOpt)

	// 初始化设备管理器
	deviceManager := device.NewDeviceManager(logger, 60, 30) // 心跳超时60秒，检查间隔30秒

	d := &Data{
		db:             client,
		rdb:            rdb,
		queue:          queueClient,
		queueServer:    queueServer,
		queueScheduler: queueScheduler,
		serverConf:     serverConf,
		conf:           appConf,
		deviceMgr:      deviceManager,
	}

	// 启动 asynq 队列服务器
	mux := asynq.NewServeMux()
	// 创建缓存服务
	cacheService := service.NewCacheService(rdb, client, logger)
	groupRepo := NewAdminGroupRepo(d, logger)
	handler.RegisterHandlers(mux, client, rdb, queueClient, appConf, cacheService, groupRepo, logger)
	go func() {
		if err := queueServer.Start(mux); err != nil {
			log.NewHelper(logger).Fatalf("Failed to start asynq server: %v", err)
		}
	}()
	log.NewHelper(logger).Info("Asynq queue server started successfully")

	go func() {
		if err := startQueueScheduler(queueScheduler, logger); err != nil {
			log.NewHelper(logger).Fatalf("Failed to start asynq scheduler: %v", err)
		}
	}()
	log.NewHelper(logger).Info("Asynq scheduler started successfully")

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if err := d.db.Close(); err != nil {
			log.NewHelper(logger).Error(err)
		}
		if err := d.rdb.Close(); err != nil {
			log.NewHelper(logger).Error(err)
		}
		if err := d.queue.Close(); err != nil {
			log.NewHelper(logger).Error(err)
		}
		// 关闭 asynq 服务器
		d.queueServer.Shutdown()
		log.NewHelper(logger).Info("asynq server stopped")
		if d.queueScheduler != nil {
			d.queueScheduler.Shutdown()
			log.NewHelper(logger).Info("asynq scheduler stopped")
		}
	}

	return d, cleanup, nil
}

// NewDeviceManager 提供设备管理器
func NewDeviceManager(d *Data) *device.DeviceManager {
	// 设置全局设备管理器实例
	deviceManagerOnce.Do(func() {
		globalDeviceManager = d.deviceMgr
	})
	return d.deviceMgr
}

// GetGlobalDeviceManager 获取全局设备管理器
func GetGlobalDeviceManager() *device.DeviceManager {
	return globalDeviceManager
}

// NewEntClient 提供 ent.Client 给需要直接访问数据库的服务
func NewEntClient(d *Data) *ent.Client {
	return d.db
}
