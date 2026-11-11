-- 02157 订阅应用启用开关（issue #37）
-- 说明：02101 会按旧结构重建 subscribe_application 表，ent 自动迁移加上的 enabled 列
-- 会在全新安装时被抹掉，因此这里用增量脚本统一补列（存量库与全新安装都覆盖）。
ALTER TABLE `subscribe_application`
ADD COLUMN `enabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用（参与客户端 UA 匹配分发；禁用后不参与分发且 UA 不进白名单）' AFTER `user_agent`;
