CREATE TABLE IF NOT EXISTS advertising_spectrum (
  ad_id         VARCHAR(255) NOT NULL,
  impression_at TIMESTAMP NOT NULL,
  click_at      TIMESTAMP
)
DISTSTYLE KEY
DISTKEY (ad_id)
;

INSERT INTO advertising_spectrum
SELECT
  ad_id,
  impression_at,
  click_at
FROM
  advertising.advertising
WHERE
      year = ?
  AND month = ?
  AND impression_at BETWEEN ? AND ?
;
