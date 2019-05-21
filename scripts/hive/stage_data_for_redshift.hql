INSERT OVERWRITE
  TABLE ${output1}
SELECT
  *
FROM
  warehouse.advertising AS advertising
WHERE
      advertising.year = ${PARTITION_YEAR}
  AND advertising.month = ${PARTITION_MONTH}
  AND advertising.impression_at BETWEEN ${TIME_START} AND ${TIME_END}
;
