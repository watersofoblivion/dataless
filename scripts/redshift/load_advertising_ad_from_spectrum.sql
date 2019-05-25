CREATE TABLE IF NOT EXISTS advertising_ad_spectrum
  DISTSTYLE KEY
  DISTKEY (ad_id)
AS
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
