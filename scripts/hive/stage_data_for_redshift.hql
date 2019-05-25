INSERT OVERWRITE
  TABLE ${output1}
SELECT
  session_id                                        AS session_id,
  user_id                                           AS user_id,
  ad_id                                             AS ad_id,
  DATE_FORMAT(impression_at, 'yyyy-MM-dd HH:mm:ss') AS impression_at,
  DATE_FORMAT(click_at, 'yyyy-MM-dd HH:mm:ss')      AS click_at
FROM
  warehouse.advertising
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= TO_UTC_TIMESTAMP(TO_DATE('${DATE_START}'), 'UTC')
  AND impression_at <  TO_UTC_TIMESTAMP(TO_DATE('${DATE_END}'),   'UTC')
;
