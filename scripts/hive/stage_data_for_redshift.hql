INSERT OVERWRITE
  TABLE ${output1}
SELECT
  session_id,
  user_id,
  ad_id,
  impression_at,
  click_at
FROM
  warehouse.advertising
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= '${DATE_START}'
  AND impression_at < '${DATE_END}'
;
