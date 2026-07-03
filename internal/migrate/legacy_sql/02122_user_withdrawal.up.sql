CREATE TABLE IF NOT EXISTS `user_withdrawal` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary Key',
    `user_id` BIGINT NOT NULL COMMENT 'User ID',
    `amount` BIGINT NOT NULL COMMENT 'Withdrawal Amount',
    `method` VARCHAR(32) DEFAULT NULL COMMENT 'Withdrawal Method',
    `content` TEXT COMMENT 'Withdrawal Content',
    `status` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Withdrawal Status',
    `reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT 'Rejection Reason',
    `processed_at` DATETIME DEFAULT NULL COMMENT 'Processed Time',
    `created_at` DATETIME NOT NULL COMMENT 'Creation Time',
    `updated_at` DATETIME NOT NULL COMMENT 'Update Time',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO `system` (`category`, `key`, `value`, `type`, `desc`, `created_at`, `updated_at`)
VALUES
    ('invite', 'WithdrawalMinAmount', '0', 'int64', 'withdrawal minimum amount', '2025-04-22 14:25:16.637', '2025-04-22 14:25:16.637'),
    ('invite', 'WithdrawalMethods', '[{"method":"alipay","label":"支付宝","enabled":true},{"method":"wechat","label":"微信","enabled":true},{"method":"usdt","label":"USDT","enabled":true}]', 'string', 'withdrawal methods', '2025-04-22 14:25:16.637', '2025-04-22 14:25:16.637');
