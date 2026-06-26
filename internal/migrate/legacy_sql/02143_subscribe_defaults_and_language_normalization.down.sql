ALTER TABLE `subscribe_application` DROP INDEX `uniq_subscribe_application_default`;
ALTER TABLE `subscribe_application` DROP COLUMN `default_unique_key`;

-- Data normalization is intentionally not reverted.
