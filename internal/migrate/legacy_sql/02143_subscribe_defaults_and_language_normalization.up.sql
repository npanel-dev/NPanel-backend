UPDATE `subscribe_application`
SET `is_default` = 0
WHERE `is_default` = 1
  AND `id` NOT IN (
    SELECT `id` FROM (
      SELECT `id`
      FROM `subscribe_application`
      WHERE `is_default` = 1
      ORDER BY `id` ASC
      LIMIT 1
    ) AS `keep_default`
  );

UPDATE `subscribe_application`
SET `is_default` = 1
WHERE NOT EXISTS (
    SELECT 1 FROM (
      SELECT `id`
      FROM `subscribe_application`
      WHERE `is_default` = 1
      LIMIT 1
    ) AS `existing_default`
  )
  AND `id` = (
    SELECT `id` FROM (
      SELECT `id`
      FROM `subscribe_application`
      ORDER BY `id` ASC
      LIMIT 1
	    ) AS `first_application`
	  );

ALTER TABLE `subscribe_application`
ADD COLUMN `default_unique_key` TINYINT
  GENERATED ALWAYS AS (CASE WHEN `is_default` = 1 THEN 1 ELSE NULL END) STORED;

ALTER TABLE `subscribe_application`
ADD UNIQUE INDEX `uniq_subscribe_application_default` (`default_unique_key`);

UPDATE `subscribe`
SET `language` = 'en-US'
WHERE LOWER(REPLACE(TRIM(`language`), '_', '-')) IN ('en', 'en-us');

UPDATE `subscribe`
SET `language` = 'zh-CN'
WHERE LOWER(REPLACE(TRIM(`language`), '_', '-')) IN ('zh', 'zh-cn', 'zh-hans', 'zh-hans-cn');

UPDATE `subscribe`
SET `language` = TRIM(`language`);

UPDATE `subscribe_category`
SET `language` = 'en-US'
WHERE LOWER(REPLACE(TRIM(`language`), '_', '-')) IN ('en', 'en-us');

UPDATE `subscribe_category`
SET `language` = 'zh-CN'
WHERE LOWER(REPLACE(TRIM(`language`), '_', '-')) IN ('zh', 'zh-cn', 'zh-hans', 'zh-hans-cn');

UPDATE `subscribe_category`
SET `language` = TRIM(`language`);
