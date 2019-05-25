WITH
  daily_events AS (
    SELECT
      actor_type                    AS actor_type,
      actor_id                      AS actor_id,
      session_id                    AS session_id,
      event_type                    AS event_type,
      object_type                   AS object_type,
      object_id                     AS object_id,
      TO_DATE(occurred_at)          AS occurred_on,
      COUNT(1)                      AS events,
      SUM(events) OVER last_7_days  AS last_7_days,
      SUM(events) OVER last_30_days AS last_30_days,
      SUM(events) OVER last_60_days AS last_60_days,
      SUM(events) OVER last_90_days AS last_90_days
    FROM
      warehouse.events
    WHERE
          year BETWEEN YEAR(DATE_SUB('${DATE_START}', 90)) AND YEAR('${DATE_END}')
      AND occurred_at >= DATE_SUB('${DATE_START}', 90)
      AND occurred_at < '${DATE_END}'
    GROUP BY
      actor_type,
      actor_id,
      session_id,
      event_type,
      object_type,
      object_id,
      occurred_on
    WINDOW
      last_7_days AS (
        PARTITION BY actor_type, actor_id, session_id, event_type, object_type, object_id
        ORDER BY occurred_on
        ROWS BETWEEN 7 PRECEDING AND CURRENT ROW
      ),
      last_30_days AS (
        PARTITION BY actor_type, actor_id, session_id, event_type, object_type, object_id
        ORDER BY occurred_on
        ROWS BETWEEN 30 PRECEDING AND CURRENT ROW
      ),
      last_60_days AS (
        PARTITION BY actor_type, actor_id, session_id, event_type, object_type, object_id
        ORDER BY occurred_on
        ROWS BETWEEN 60 PRECEDING AND CURRENT ROW
      ),
      last_90_days AS (
        PARTITION BY actor_type, actor_id, session_id, event_type, object_type, object_id
        ORDER BY occurred_on
        ROWS BETWEEN 90 PRECEDING AND CURRENT ROW
      )
  )
SELECT *
FROM daily_events
LIMIT 10
;
