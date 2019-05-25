DESCRIBE ${output1};

WITH
  events AS (
    SELECT
      ad_id                  AS ad_id,
      TO_DATE(impression_at) AS day,
      click_at IS NOT NULL   AS clicked
    FROM
      warehouse.advertising
    WHERE
          year = ${PARTITION_YEAR}
      AND month = ${PARTITION_MONTH}
      AND impression_at >= '${DATE_START} 00:00:00'
      AND impression_at < '${DATE_END} 00:00:00'
  ),
  impressions AS (
    SELECT
      ad_id    AS ad_id,
      day      AS day,
      COUNT(1) AS count
    FROM events
    WHERE NOT clicked
    GROUP BY ad_id, day
  ),
  clicks AS (
    SELECT
      ad_id    AS ad_id,
      day      AS day,
      COUNT(1) AS count
    FROM events
    WHERE clicked
    GROUP BY ad_id, day
  )
INSERT OVERWRITE
  TABLE ${output1}
SELECT
  impressions.ad_id                                                AS ad_id,
  DATE_FORMAT(impressions.day, "yyyy-MM-dd")                       AS day,
  impressions.count                                                AS impressions,
  clicks.count                                                     AS clicks,
  (CAST(clicks.count AS float) / CAST(impressions.count AS float)) AS clickthrough_rate
FROM
  impressions
JOIN clicks
  ON  impressions.day = clicks.day
  AND impressions.ad_id = clicks.ad_id
;
