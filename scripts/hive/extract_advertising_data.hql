WITH
  events AS (
    SELECT
      events.session_id  AS session_id,
      events.context_id  AS context_id,
      events.event_id    AS event_id,
      events.object_id   AS ad_id,
      events.occurred_at AS impression_at
    FROM
      warehouse.events AS events
    WHERE
          events.year = ${PARTITION_YEAR}
      AND events.month = ${PARTITION_MONTH}
      AND events.impression_at BETWEEN ${TIME_START} AND ${TIME_END}
      AND object_type = 'ad'
      AND event IN ('impression', 'click')
  ),
  impressions AS (
    SELECT *
    FROM events
    WHERE event = 'impression'
  ),
  clicks AS (
    SELECT *
    FROM events
    WHERE event = 'click'
  )
INSERT INTO
  TABLE    warehouse.advertising
  PARTITON (year = ${PARTITION_YEAR}, month = ${PARTITION_MONTH})
SELECT
  impressions.session_id    AS session_id,
  impressions.user_id       AS user_id,
  impressions.ad_id         AS ad_id,
  impressions.impression_at AS impression_at,
  clicks.click_at           AS click_at
FROM
  impressions
LEFT OUTER JOIN clicks
  ON  impressions.session_id = clicks.session_id
  AND impressions.context_id = clicks.context_id
  AND impressions.user_id    = clicks.user_id
  AND impressions.ad_id      = clicks.ad_id
  AND impressions.event_id   = clicks.parent_id
;
