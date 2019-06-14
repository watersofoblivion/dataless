INSERT INTO
  TABLE     advertising.advertising
  PARTITION (year = ${PARTITION_YEAR}, month)
SELECT
  impressions.ad_id       AS ad_id,
  impressions.occurred_at AS impression_at,
  clicks.occurred_at      AS click_at,
  ${PARTITION_MONTH}      AS month
FROM
  advertising.impressions
LEFT OUTER JOIN advertising.clicks
  ON  impressions.ad_id         = clicks.ad_id
  AND impressions.impression_id = clicks.impression_id
;
