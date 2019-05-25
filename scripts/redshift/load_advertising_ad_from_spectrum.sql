CREATE TABLE IF NOT EXISTS advertising_ad_spectrum (
  session_id    VARCHAR(255) NOT NULL,
  user_id       VARCHAR(255) NOT NULL,
  ad_id         VARCHAR(255) NOT NULL,
  impression_at TIMESTAMP NOT NULL,
  click_at      TIMESTAMP
)
DISTSTYLE KEY
DISTKEY (ad_id)
;

INSERT INTO advertising_ad_spectrum
SELECT
  session_id,
  user_id,
  ad_id,
  impression_at,
  click_at
FROM
  lake.advertising
WHERE
      year = ?
  AND month = ?
  AND impression_at BETWEEN ? AND ?
;
