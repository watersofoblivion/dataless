CREATE TABLE advertising_user_spectrum
  DISTSTYLE KEY
  DISTKEY (user_id)
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
