CREATE TABLE IF NOT EXISTS `subscribe_rebate_config` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '返佣配置ID',
    `subscribe_id` bigint NOT NULL COMMENT '订阅套餐ID',
    `price_option_id` bigint NOT NULL DEFAULT 0 COMMENT '价格档位ID，0表示套餐整体默认',
    `referral_percentage` tinyint NOT NULL DEFAULT 0 COMMENT '返佣百分比 0-100',
    `enabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用，false表示显式关闭该档返佣（不回退兜底）',
    `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_subscribe_price_option` (`subscribe_id`, `price_option_id`),
    KEY `idx_subscribe_id` (`subscribe_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='订阅套餐返佣比例配置';
