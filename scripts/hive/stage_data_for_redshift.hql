!echo "PARTITION_YEAR: ${PARTITION_YEAR}"
!echo "PARTITION_MONTH: ${PARTITION_MONTH}"
!echo "DATE_START: ${DATE_START}"
!echo "DATE_END: ${DATE_END}"
!echo "output1: ${output1}"

!echo "INSERT OVERWRITE TABLE ${output1} SELECT session_id, user_id, ad_id, impression_at, click_at FROM warehouse.advertising WHERE year = ${PARTITION_YEAR} AND month = ${PARTITION_MONTH} AND impression_at >= '${DATE_START} 00:00:00' AND impression_at < '${DATE_END} 00:00:00'"

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
  AND impression_at >= '${DATE_START} 00:00:00'
  AND impression_at < '${DATE_END} 00:00:00'
;
