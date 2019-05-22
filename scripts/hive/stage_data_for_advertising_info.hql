INSERT OVERWRITE
  TABLE ${output1}
WITH
  events (
    SELECT
      ad_id                            AS ad_id,
      DATE_TRUNC('day', impression_at) AS day,
      click_at IS NOT NULL             AS clicked
    FROM
      warehouse.advertising
    WHERE
          year = ${PARTITION_YEAR}
      AND month = ${PARTITION_MONTH}
      AND impression_at >= '${TIME_START}'
      AND impression_at < '${TIME_END}'
  ),
  impressions (
    SELECT
      ad_id    AS ad_id,
      day      AS day,
      COUNT(1) AS count
    FROM events
    WHERE clicked
    GROUP BY day
  ),
  clicks AS (
    SELECT
      ad_id    AS ad_id,
      day      AS day,
      COUNT(1) AS count
    FROM events
    WHERE NOT clicked
    GROUP BY day
  )
SELECT
  impressions.ad_id                                AS ad_id,
  impressions.day                                  AS day,
  impressions.count                                AS impressions,
  clicks.count                                     AS clicks,
  (clicks.count::float / impressions.count::float) AS clickthrough_rate
FROM
  impressions
JOIN clicks
  ON  impressions.day = clicks.day
  AND impressions.ad_id = clicks.ad_id
;
