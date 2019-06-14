CREATE TABLE IF NOT EXISTS raw_impressions_spectrum (
  impression_id VARCHAR(255) NOT NULL,
  ad_id         VARCHAR(255) NOT NULL,
  occurred_at   TIMESTAMP
)
DISTSTYLE KEY
DISTKEY (ad_id)
;

INSERT INTO raw_impressions_spectrum
SELECT
  impression_id,
  ad_id,
  occurred_at
FROM
  advertising.raw_impressions
WHERE
      year = ?
  AND month = ?
  AND impression_at BETWEEN ? AND ?
;
