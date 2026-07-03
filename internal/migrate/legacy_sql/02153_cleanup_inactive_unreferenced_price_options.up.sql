DELETE p
FROM `subscribe_price_option` p
LEFT JOIN `order` o ON o.`price_option_id` = p.`id`
WHERE (p.`show` = 0 OR p.`sell` = 0)
  AND o.`id` IS NULL;
