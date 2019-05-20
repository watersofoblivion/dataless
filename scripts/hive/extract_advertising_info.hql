INSERT OVERWRITE
  TABLE ${output1}
SELECT

FROM
  warehouse.advertising
WHERE
      warehouse.advertising.year = ${PARTITION_YEAR}
  AND warehouse.advertising.month = ${PARTITION_MONTH}
  AND warehouse.advertising.impression_at BETWEEN ${TIME_START} AND ${TIME_END}
;
