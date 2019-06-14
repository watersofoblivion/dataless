CREATE TABLE IF NOT EXISTS raw_clicks_spectrum (
  click_id      VARCHAR(255) NOT NULL,
  impression_id VARCHAR(255) NOT NULL,
  ad_id         VARCHAR(255) NOT NULL,
  occurred_at   TIMESTAMP
)
DISTSTYLE KEY
DISTKEY (ad_id)
;

INSERT INTO raw_clicks_spectrum
SELECT
  click_id                 AS click_id,
  impression_id            AS impression_id,
  ad_id                    AS ad_id,
  occurred_at :: timestamp AS occurred_at
FROM
  advertising.raw_clicks
WHERE
      partition_0 = ?
  AND partition_1 = ?
  AND partition_2 = ?
  AND occurred_at BETWEEN ? AND ?
;
