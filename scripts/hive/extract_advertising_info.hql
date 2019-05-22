INSERT OVERWRITE
  TABLE ${output1}
WITH
  events (
    SELECT
      advertising.ad_id                            AS ad_id,
      DATE_TRUNC('day', advertising.impression_at) AS day,
      click_at IS NOT NULL                         AS clicked
    FROM
      warehouse.advertising AS advertising
    WHERE
          advertising.year = ${PARTITION_YEAR}
      AND advertising.month = ${PARTITION_MONTH}
      AND advertising.impression_at >= '${TIME_START}'
      AND advertising.impression_at < '${TIME_END}'
  ),
  impressions (
    SELECT
      events.ad_id AS ad_id,
      events.day   AS day,
      COUNT(1)     AS count
    FROM events
    WHERE advertising.clicked
    GROUP BY events.day
  ),
  clicks AS (
    SELECT
      events.ad_id AS ad_id,
      events.day   AS day,
      COUNT(1)     AS count
    FROM events
    WHERE NOT advertising.clicked
    GROUP BY events.day
  )
SELECT
  impressions.ad_id                                AS ad_id,
  impressions.day                                  AS day,
  impressions.count                                AS impressions,
  clicks.count                                     AS clicks,
  (clicks.count::float / impressions.count::float) AS clickthrough_rate
FROM
  impressions AS impressions
JOIN
  clicks
    ON  impressions.day = clicks.day
    AND impressions.ad_id = clicks.ad_id
;
