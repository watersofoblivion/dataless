INSERT OVERWRITE
  TABLE ${output1}
SELECT
  *
FROM
  warehouse.advertising
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= '${TIME_START}'
  AND impression_at < '${TIME_END}'
;
