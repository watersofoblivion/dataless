INSERT OVERWRITE
  TABLE ${output1}
SELECT
  impression_id,
  ad_id,
  occurred_at
FROM
  advertising.impressions
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND occurred_at >= '${DATE_START} 00:00:00'
  AND occurred_at <  '${DATE_END} 00:00:00'
;
