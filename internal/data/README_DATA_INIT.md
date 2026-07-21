# 数据初始化文档

## 概述

本项目提供了完整的数据初始化系统，用于在首次部署或重置环境时创建必要的默认数据。

## 功能特性

### 1. 自动数据初始化
- 服务启动时自动执行数据初始化
- 智能检测已存在数据，避免重复初始化
- 新库或 legacy 库初始化失败会阻止服务启动

### 2. 初始化的数据类型

#### 认证方法 (Auth Methods)
- **邮件认证**: SMTP 配置，支持邮箱验证和通知；四个 `*_template` 由 `ensureEmailAuthMethodTemplates` 从 `pkg/email/template.go` 回填（legacy SQL 骨架为空）
- **手机认证**: 使用abosend短信平台
- **OAuth认证**: Apple、Google、GitHub登录
- **设备认证**: 设备管理和安全配置
- **Telegram认证**: Telegram机器人集成

#### 支付方式 (Payment Methods)
- **余额支付**: 默认启用
- **支付宝**: 需要配置商户信息
- **微信支付**: 需要配置商户信息
- **Stripe**: 国际信用卡支付

#### 系统配置 (System Configuration)
- **站点配置**: 站点名称、描述、Logo等
- **货币配置**: 默认货币和符号
- **注册配置**: 注册限制、试用设置
- **邀请配置**: 推荐奖励设置
- **验证配置**: Cloudflare Turnstile集成
- **订阅配置**: 订阅模式和路径设置
- **节点配置**: 推送间隔和流量报告设置

#### 默认公告 (Announcements)
- 由 legacy SQL seed（`00002_init_basic_data.up.sql`）提供

#### 默认文档 (Documents)
- 由 legacy SQL seed 提供

## 使用方法

### 1. 自动初始化（推荐）
服务启动时会自动执行数据初始化：
```bash
# 正常启动服务
./bin/npanel -conf ./configs
```

### 2. 手动数据初始化
默认数据在**服务启动时**由 `internal/migrate` 自动写入。MySQL legacy 库保留旧 SQL seed 兼容路径，新 MySQL/PostgreSQL 库使用 portable baseline。`make seed` 仅打印提示，不会单独执行 seed：

```bash
make dev
# 或
go run ./cmd/npanel -conf ./configs
```

### 3. 完整数据库迁移
新库首次启动走 `AutoMigrateWithData`（结构迁移 + 基础数据）。`make migrate` 同样仅为提示：

```bash
make migrate
# 实际迁移请启动服务：
go run ./cmd/npanel -conf ./configs
```

### 4. 数据备份和导出
当前仓库未提供独立导出命令，请使用数据库原生备份工具（如 `mysqldump` 或 `pg_dump`）备份重要数据。

## 配置文件

确保 `configs/config.yaml` 中的数据库配置正确：
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

PostgreSQL 示例：

```yaml
data:
  database:
    driver: postgres
    source: postgres://npanel:password@127.0.0.1:5432/npanel?sslmode=disable
```

## 注意事项

### 1. 生产环境
- 生产环境部署前请检查并更新默认配置
- 修改默认的短信平台配置
- 更新支付方式的商户信息
- 设置正确的站点信息和Logo

### 2. 数据安全
- 默认初始化的配置包含测试用的API密钥
- 请在生产环境中替换为真实的API密钥
- 定期备份重要数据

### 3. 权限设置
- 确保数据库用户有创建表的权限
- 确保Redis服务正常运行
- 检查文件系统权限

## 故障排除

### 1. 数据库连接失败
- 检查数据库服务状态
- 验证连接字符串和认证信息
- 确认数据库用户权限

### 2. Redis连接失败
- 检查Redis服务状态
- 验证连接配置
- 检查网络连接

### 3. 初始化失败
- 查看服务日志获取详细错误信息
- 检查配置文件格式
- 确认依赖服务（数据库、Redis）正常

### 4. 数据重复
- 系统会自动检测已存在的数据
- 如需重新初始化，请先清空相关表
- 或者手动删除特定配置记录

## 扩展自定义数据

在 `internal/migrate/migrate.go` 的 `InitBasicData` 中增加初始化步骤，或扩展 `initLegacyDefaultData` 所用的 embedded SQL。
