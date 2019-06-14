CREATE TABLE IF NOT EXISTS clicks_spectrum (
  click_id      VARCHAR(255) NOT NULL,
  impression_id VARCHAR(255) NOT NULL,
  ad_id         VARCHAR(255) NOT NULL,
  occurred_at   TIMESTAMP
)
DISTSTYLE KEY
DISTKEY (ad_id)
;

INSERT INTO clicks_spectrum
SELECT
  click_id,
  impression_id,
  ad_id,
  occurred_at
FROM
  advertising.clicks
WHERE
      year = ?
  AND month = ?
  AND occurred_at BETWEEN ? AND ?
;
