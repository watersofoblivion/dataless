WITH
  events AS (
    SELECT
      *
    FROM
      warehouse.events
    WHERE
          year = ${PARTITION_YEAR}
      AND month = ${PARTITION_MONTH}
      AND occurred_at >= '${TIME_START}'
      AND occurred_at < '${TIME_END}'
      AND actor_type = 'customer'
      AND event_type IN ('impression', 'click')
      AND object_type = 'ad'
  ),
  impressions AS (SELECT * FROM events WHERE events.event_type = 'impression'),
  clicks      AS (SELECT * FROM events WHERE events.event_type = 'click')
INSERT INTO
  TABLE     warehouse.advertising
  PARTITION (year = ${PARTITION_YEAR}, month = ${PARTITION_MONTH})
SELECT
  impressions.session_id  AS session_id,
  impressions.actor_id    AS user_id,
  impressions.object_id   AS ad_id,
  impressions.occurred_at AS impression_at,
  clicks.occurred_at      AS click_at
FROM
  impressions
LEFT OUTER JOIN clicks
  ON  impressions.session_id = clicks.session_id
  AND impressions.context_id = clicks.context_id
  AND impressions.actor_id   = clicks.actor_id
  AND impressions.object_id  = clicks.object_id
  AND impressions.event_id   = clicks.parent_id
;
