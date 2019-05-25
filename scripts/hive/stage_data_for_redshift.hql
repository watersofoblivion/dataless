DESCRIBE ${output1};

INSERT OVERWRITE
  TABLE ${output1}
SELECT
  session_id                                    AS session_id,
  user_id                                       AS user_id,
  ad_id                                         AS ad_id,
  TO_CHAR(impression_at, 'yyyy-MM-dd HH:mm:ss') AS impression_at,
  TO_CHAR(click_at, 'yyyy-MM-dd HH:mm:ss')      AS click_at
FROM
  warehouse.advertising
WHERE
      year = ${PARTITION_YEAR}
  AND month = ${PARTITION_MONTH}
  AND impression_at >= '${DATE_START} 00:00:00'
  AND impression_at < '${DATE_END} 00:00:00'
;
