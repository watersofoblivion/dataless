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
  AND month = ?
  AND impression_at BETWEEN ? AND ?
;
