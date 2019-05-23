CREATE TABLE advertising_user_spectrum
  DISTSTYLE KEY
  DISTKEY (user_id)
AS
SELECT
  *
FROM
  warehouse.advertising
WHERE
      year = $1
  AND year = $2
  AND impression_at BETWEEN $3 AND $4
;
