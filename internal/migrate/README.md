# 数据库迁移系统

## 概述

本项目的数据库迁移系统完全基于Ent ORM实现，提供了完整的数据库结构迁移和默认数据初始化功能。

## 特性

### 1. 基于Ent ORM
- 使用Ent的类型安全查询构建器
- 支持事务操作
- 避免SQL注入风险
- 跨数据库兼容


### 3. 智能初始化
- 自动检测已存在数据
- 避免重复初始化
- 支持增量数据更新

### 4. 完整的数据初始化
- 认证方法配置
- 支付方式设置
- 系统配置
- 默认管理员账户
- 默认公告和文档

## 迁移组件

### 1. Migrator (`migrate.go`)
核心迁移器，包含以下功能：
- `AutoMigrate()`: 仅数据库结构迁移
- `AutoMigrateWithData()`: 完整迁移（结构+数据）
- `InitBasicData()`: 初始化 MySQL legacy 基础数据（legacy SQL + 邮件模板回填）
- portable baseline：MySQL/PostgreSQL 新库共用的 Ent 基础数据初始化
- `ensureEmailAuthMethodTemplates()`: 邮件认证空模板幂等回填
- `CreateDefaultAdminUser()`: 创建默认管理员

### 2. 修复工具 (`repair.go`)
数据修复工具，用于数据迁移后的修复和清理。

## 使用方法

### 1. 自动迁移（服务启动时）
服务启动时在 `internal/data/data.go` 中调用 migrator：

```go
migrator := migrate.NewMigrator(client, logger, appConf, c.Database.Driver, c.Database.Source)
// MySQL/PostgreSQL 新库: AutoMigrateWithData；已有 MySQL legacy 库: 兼容迁移 + InitBasicData
```

### 2. 手动迁移

#### 基础数据同步
```bash
# 启动服务即会执行 InitBasicData（含邮件模板回填）
make dev
# 或
go run ./cmd/npanel -conf ./configs
```

#### 完整迁移
```bash
make migrate   # 提示：新库首次启动时自动完成结构+数据迁移
```

#### 迁移验证
双方言迁移集成测试可通过 `NPANEL_TEST_MYSQL_DSN` 与 `NPANEL_TEST_POSTGRES_DSN` 启用：

```bash
NPANEL_TEST_MYSQL_DSN='...' NPANEL_TEST_POSTGRES_DSN='...' \
  go test ./internal/migrate -run TestPortableBaselineAgainstDatabase -v -count=1
```

## 默认数据详情

### 1. 认证方法
- **邮件认证**: SMTP 配置框架；默认 HTML 模板来自 `pkg/email/template.go`，启动时由 `ensureEmailAuthMethodTemplates` 回填；运行时 `EmailAuthConfig.Unmarshal` 亦会兜底空模板
- **手机认证**: abosend短信平台（包含测试密钥）
- **OAuth认证**: Apple、Google、GitHub配置
- **设备认证**: 设备管理配置
- **Telegram认证**: 机器人集成配置

### 2. 支付方式
- **余额支付**: 默认启用
- **支付宝**: 配置框架
- **微信支付**: 配置框架

### 3. 系统配置
- 站点基本信息（名称、Logo、关键词等）
- 货币配置（默认货币和符号）
- 注册限制和试用设置
- 邀请推荐系统配置
- 验证服务配置
- 订阅模式配置

### 4. 默认管理员
管理员信息从配置文件读取，如果没有配置则使用默认值：
- **邮箱**: 从config.yaml读取，默认admin@example.com
- **密码**: 从config.yaml读取，默认admin123456
- **算法**: 从config.yaml读取，默认default
- **权限**: 超级管理员

配置示例：
```yaml
app:
  admin:
    email: "your-admin@example.com"
    password: "your-secure-password"
    algo: "default"
```


## 配置文件要求

确保 `configs/config.yaml` 包含正确的数据库配置：

```yaml
data:
  database:
    driver: mysql
    source: root:password@tcp(127.0.0.1:3306)/npanel_pro?parseTime=True&loc=Local
  redis:
    addr: 127.0.0.1:6379
    password:
    read_timeout: 0.2s
    write_timeout: 0.2s
    db: 0
    pool_size: 10
    min_idle_conns: 5
```

PostgreSQL 使用 `driver: postgres` 和标准连接 URL，例如：

```yaml
data:
  database:
    driver: postgres
    source: postgres://npanel:password@127.0.0.1:5432/npanel?sslmode=disable
```

## 生产环境注意事项

### 1. 安全配置
- 更改默认管理员密码
- 更新短信平台API密钥
- 配置真实的支付商户信息

### 2. 数据备份
- 迁移前备份现有数据
- 请使用数据库原生备份工具（如 `mysqldump` 或 `pg_dump`）备份重要配置

### 3. 性能考虑
- 大量数据迁移时建议分批执行
- 监控迁移过程中的内存使用

## 故障排除

### 1. 连接失败
```
Error: failed to create data client
解决: 检查数据库服务状态和连接配置
```

### 2. 权限问题
```
Error: failed to create schema
解决: 确保数据库用户有CREATE TABLE权限
```

### 3. 数据重复
```
Warning: Auth methods already exist
解决: 这是正常提示，系统会跳过已存在数据
```

### 4. 事务失败
```
Error: failed to commit transaction
解决: 检查数据库磁盘空间和锁定状态
```

## 开发和扩展

### 1. 添加新的初始化数据
在 `migrate.go` 中的 `InitBasicData` 方法中添加新的初始化函数：

```go
// 初始化自定义数据
func (m *Migrator) initCustomData(ctx context.Context, tenantID int64) error {
    // 检查是否已存在
    count, err := m.client.ProxyCustomTable.Query().
        Where(proxycustomtable.TenantID(tenantID)).
        Count(ctx)
    if err != nil {
        return err
    }

    if count > 0 {
        return nil
    }

    // 创建数据
    _, err = m.client.ProxyCustomTable.Create().
        SetTenantID(tenantID).
        SetName("Custom Data").
        Save(ctx)
    return err
}
```

### 2. 自定义迁移逻辑
可以通过实现自定义的Migrator接口来扩展迁移功能：

```go
type CustomMigrator interface {
    Migrate(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

### 3. 数据验证
迁移完成后，请启动服务并检查日志，或查询数据库确认初始化数据完整性。

这个迁移系统确保了数据库结构的一致性和初始数据的完整性，为项目的部署和维护提供了可靠的解决方案。
