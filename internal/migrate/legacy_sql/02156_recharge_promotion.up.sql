-- 充值促销活动主表
CREATE TABLE IF NOT EXISTS `recharge_promotion` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '活动ID',
    `name` varchar(64) NOT NULL COMMENT '活动名称',
    `description` text COMMENT '活动描述 Markdown',
    `start_time` bigint NOT NULL DEFAULT 0 COMMENT '开始时间 Unix毫秒 0=不限',
    `end_time` bigint NOT NULL DEFAULT 0 COMMENT '结束时间 Unix毫秒 0=长期',
    `status` tinyint(1) NOT NULL DEFAULT 0 COMMENT '启用状态 0=停用 1=启用',
    `priority` int NOT NULL DEFAULT 0 COMMENT '优先级 DESC',
    `stack_mode` tinyint NOT NULL DEFAULT 0 COMMENT '叠加策略 0=互斥 1=叠加',
    `custom_bonus_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否对自定义金额赠送',
    `custom_bonus_percent` int NOT NULL DEFAULT 0 COMMENT '自定义金额赠送比例 万分比 0-10000',
    `sort` int NOT NULL DEFAULT 0 COMMENT '列表排序',
    `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_status_time` (`status`, `start_time`, `end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='充值促销活动';

-- 充值促销档位子表
CREATE TABLE IF NOT EXISTS `recharge_promotion_tier` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '档位ID',
    `promotion_id` bigint NOT NULL COMMENT '所属活动ID',
    `amount` bigint NOT NULL DEFAULT 0 COMMENT '充值档位金额 分',
    `bonus` bigint NOT NULL DEFAULT 0 COMMENT '赠送金额 分',
    `label` varchar(32) NOT NULL DEFAULT '' COMMENT '档位标签',
    `sort` int NOT NULL DEFAULT 0 COMMENT '档位排序 ASC',
    `enable` tinyint(1) NOT NULL DEFAULT 1 COMMENT '单档启停',
    `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_promotion` (`promotion_id`),
    KEY `idx_promotion_amount_enable` (`promotion_id`, `amount`, `enable`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='充值促销档位';
