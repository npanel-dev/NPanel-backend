ALTER TABLE `order` DROP INDEX IF EXISTS `idx_price_option_id`;
ALTER TABLE `order` DROP COLUMN IF EXISTS `option_price`;
ALTER TABLE `order` DROP COLUMN IF EXISTS `duration_value`;
ALTER TABLE `order` DROP COLUMN IF EXISTS `duration_unit`;
ALTER TABLE `order` DROP COLUMN IF EXISTS `price_option_name`;
ALTER TABLE `order` DROP COLUMN IF EXISTS `price_option_id`;
DROP TABLE IF EXISTS `subscribe_price_option`;
