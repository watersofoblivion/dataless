CREATE TABLE advertising_user_spectrum
  DISTSTYLE KEY
  DISTKEY (user_id)
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
