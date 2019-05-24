CREATE TABLE advertising_ad_spectrum
  DISTSTYLE KEY
  DISTKEY (ad_id)
AS
SELECT
  *
FROM
  warehouse.advertising
WHERE
      year = ?
  AND year = ?
  AND impression_at BETWEEN ? AND ?
;
