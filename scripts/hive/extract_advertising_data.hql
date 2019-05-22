WITH
  events AS (
    SELECT
      events.session_id  AS session_id,
      events.context_id  AS context_id,
      events.parent_id   AS parent_id,
      events.actor_id    AS actor_id,
      events.event_type  AS event_type,
      events.event_id    AS event_id,
      events.object_id   AS object_id,
      events.occurred_at AS occurred_at
    FROM
      warehouse.events AS events
    WHERE
          events.year = ${PARTITION_YEAR}
      AND events.month = ${PARTITION_MONTH}
      AND events.occurred_at >= '${TIME_START}'
      AND events.occurred_at < '${TIME_END}'
      AND events.actor_type = 'customer'
      AND events.event_type IN ('impression', 'click')
      AND events.object_type = 'ad'
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
