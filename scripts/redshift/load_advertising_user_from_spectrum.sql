CREATE TABLE advertising_user_spectrum
  DISTSTYLE KEY
  DISTKEY (user_id)
AS
SELECT
  *
FROM
  warehouse.advertising AS advertising
WHERE
      advertising.year = $1
  AND advertising.year = $2
  AND advertising.impression_at BETWEEN $3 AND $4
;
