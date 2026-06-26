CREATE TABLE IF NOT EXISTS `subscribe_price_option` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT 'Subscribe Price Option ID',
    `subscribe_id` bigint NOT NULL COMMENT 'Subscribe ID',
    `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Option Name',
    `duration_unit` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'Month' COMMENT 'Duration Unit',
    `duration_value` bigint NOT NULL DEFAULT 1 COMMENT 'Duration Value',
    `price` bigint NOT NULL DEFAULT 0 COMMENT 'Price',
    `original_price` bigint NOT NULL DEFAULT 0 COMMENT 'Original Price',
    `inventory` int NOT NULL DEFAULT -1 COMMENT 'Inventory',
    `show` tinyint(1) NOT NULL DEFAULT 1 COMMENT 'Show',
    `sell` tinyint(1) NOT NULL DEFAULT 1 COMMENT 'Sell',
    `is_default` tinyint(1) NOT NULL DEFAULT 0 COMMENT 'Is Default',
    `sort` int NOT NULL DEFAULT 0 COMMENT 'Sort Order',
    `created_at` datetime(3) DEFAULT NULL COMMENT 'Create Time',
    `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update Time',
    PRIMARY KEY (`id`),
    KEY `idx_subscribe_id` (`subscribe_id`),
    KEY `idx_subscribe_sell_sort` (`subscribe_id`, `sell`, `sort`),
    KEY `idx_subscribe_show_sort` (`subscribe_id`, `show`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Subscribe Price Options';

SET @column_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND COLUMN_NAME = 'price_option_id');
SET @sql = IF(@column_exists = 0,
    'ALTER TABLE `order` ADD COLUMN `price_option_id` bigint NOT NULL DEFAULT 0 COMMENT ''Subscribe Price Option ID'' AFTER `subscribe_id`',
    'SELECT ''Column price_option_id already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND COLUMN_NAME = 'price_option_name');
SET @sql = IF(@column_exists = 0,
    'ALTER TABLE `order` ADD COLUMN `price_option_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '''' COMMENT ''Price Option Name Snapshot'' AFTER `price_option_id`',
    'SELECT ''Column price_option_name already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND COLUMN_NAME = 'duration_unit');
SET @sql = IF(@column_exists = 0,
    'ALTER TABLE `order` ADD COLUMN `duration_unit` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '''' COMMENT ''Duration Unit Snapshot'' AFTER `price_option_name`',
    'SELECT ''Column duration_unit already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND COLUMN_NAME = 'duration_value');
SET @sql = IF(@column_exists = 0,
    'ALTER TABLE `order` ADD COLUMN `duration_value` bigint NOT NULL DEFAULT 0 COMMENT ''Duration Value Snapshot'' AFTER `duration_unit`',
    'SELECT ''Column duration_value already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND COLUMN_NAME = 'option_price');
SET @sql = IF(@column_exists = 0,
    'ALTER TABLE `order` ADD COLUMN `option_price` bigint NOT NULL DEFAULT 0 COMMENT ''Price Option Snapshot'' AFTER `duration_value`',
    'SELECT ''Column option_price already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'order' AND INDEX_NAME = 'idx_price_option_id');
SET @sql = IF(@index_exists = 0,
    'ALTER TABLE `order` ADD INDEX `idx_price_option_id` (`price_option_id`)',
    'SELECT ''Index idx_price_option_id already exists in order table''');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
