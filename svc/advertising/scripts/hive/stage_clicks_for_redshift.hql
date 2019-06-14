INSERT OVERWRITE
  TABLE ${output1}
SELECT
  click_id,
  impression_id,
  ad_id,
  occurred_at
FROM
  advertising.clicks
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= '${DATE_START} 00:00:00'
  AND impression_at <  '${DATE_END} 00:00:00'
;
