CREATE TABLE advertising_ad_spectrum
  DISTSTYLE KEY
  DISTKEY (ad_id)
AS
SELECT
  *
FROM
  lake.advertising
WHERE
      year = ?
  AND month = ?
  AND impression_at BETWEEN ? AND ?
;
