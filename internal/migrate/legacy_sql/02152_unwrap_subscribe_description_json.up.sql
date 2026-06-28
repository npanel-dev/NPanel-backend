UPDATE `subscribe`
SET
    `short_description` = CASE
        WHEN (`short_description` IS NULL OR TRIM(`short_description`) = '')
        THEN NULLIF(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.description')), '')
        ELSE `short_description`
    END,
    `features` = CASE
        WHEN (`features` IS NULL OR TRIM(`features`) = '') AND JSON_EXTRACT(`description`, '$.features') IS NOT NULL
        THEN JSON_EXTRACT(`description`, '$.features')
        ELSE `features`
    END,
    `detail_format` = CASE
        WHEN LOWER(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.detail_format'))) IN ('markdown', 'html', 'text', 'rich')
        THEN CASE
            WHEN LOWER(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.detail_format'))) = 'rich' THEN 'html'
            ELSE LOWER(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.detail_format')))
        END
        ELSE `detail_format`
    END,
    `detail_content` = CASE
        WHEN (`detail_content` IS NULL OR TRIM(`detail_content`) = '')
        THEN COALESCE(
            NULLIF(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.detail_content')), ''),
            NULLIF(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.content')), '')
        )
        ELSE `detail_content`
    END,
    `description` = COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(`description`, '$.description')), ''), '')
WHERE JSON_VALID(`description`) = 1
  AND JSON_TYPE(JSON_EXTRACT(`description`, '$')) = 'OBJECT';

UPDATE `subscribe`
SET
    `features` = CASE
        WHEN (`features` IS NULL OR TRIM(`features`) = '') THEN `description`
        ELSE `features`
    END,
    `description` = ''
WHERE JSON_VALID(`description`) = 1
  AND JSON_TYPE(JSON_EXTRACT(`description`, '$')) = 'ARRAY';
