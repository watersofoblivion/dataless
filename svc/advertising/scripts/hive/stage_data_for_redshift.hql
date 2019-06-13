INSERT OVERWRITE
  TABLE ${output1}
SELECT
  ad_id,
  impression_at,
  click_at
FROM
  advertising.advertising
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= '${DATE_START} 00:00:00'
  AND impression_at <  '${DATE_END} 00:00:00'
CLUSTER BY 1
;
