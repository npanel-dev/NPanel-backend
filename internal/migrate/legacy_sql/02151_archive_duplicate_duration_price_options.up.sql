UPDATE `subscribe_price_option` p
JOIN (
    SELECT p1.`id`
    FROM `subscribe_price_option` p1
    JOIN `subscribe_price_option` p2
      ON p2.`subscribe_id` = p1.`subscribe_id`
     AND p2.`option_type` = p1.`option_type`
     AND p2.`duration_unit` = p1.`duration_unit`
     AND p2.`duration_value` = p1.`duration_value`
     AND p2.`id` <> p1.`id`
    WHERE p1.`option_type` = 'duration'
      AND p1.`show` = 1
      AND p1.`sell` = 1
      AND p2.`show` = 1
      AND p2.`sell` = 1
      AND (
          p2.`is_default` > p1.`is_default`
          OR (p2.`is_default` = p1.`is_default` AND p2.`sort` > p1.`sort`)
          OR (p2.`is_default` = p1.`is_default` AND p2.`sort` = p1.`sort` AND p2.`id` < p1.`id`)
      )
) duplicate_options ON duplicate_options.`id` = p.`id`
SET p.`show` = 0,
    p.`sell` = 0,
    p.`is_default` = 0,
    p.`version` = p.`version` + 1,
    p.`updated_at` = NOW();
