WITH
  events AS (
    SELECT
      session_id,
      context_id,
      parent_id,
      actor_id,
      event_type,
      event_id,
      object_id,
      occurred_at
    FROM
      warehouse.events
    WHERE
          year = ${PARTITION_YEAR}
      AND month = ${PARTITION_MONTH}
      AND occurred_at >= '${DATE_START} 00:00:00'
      AND occurred_at < '${DATE_END} 00:00:00'
      AND actor_type = 'customer'
      AND event_type IN ('impression', 'click')
      AND object_type = 'ad'
  ),
  impressions AS (SELECT * FROM events WHERE event_type = 'impression'),
  clicks      AS (SELECT * FROM events WHERE event_type = 'click')
INSERT INTO
  TABLE     warehouse.advertising
  PARTITION (year = ${PARTITION_YEAR}, month)
SELECT
  impressions.session_id  AS session_id,
  impressions.actor_id    AS user_id,
  impressions.object_id   AS ad_id,
  impressions.occurred_at AS impression_at,
  clicks.occurred_at      AS click_at,
  ${PARTITION_MONTH}      AS month
FROM
  impressions
LEFT OUTER JOIN clicks
  ON  impressions.session_id = clicks.session_id
  AND impressions.context_id = clicks.context_id
  AND impressions.actor_id   = clicks.actor_id
  AND impressions.object_id  = clicks.object_id
  AND impressions.event_id   = clicks.parent_id
;
