CREATE EXTERNAL SCHEMA lake
FROM DATA CATALOG
DATABASE 'warehouse'
IAM_ROLE ''
CREATE EXTERNAL DATABASE
IF NOT EXISTS;

CREATE EXTERNAL TABLE warehouse.advertising (
  session_id    VARCHAR(64) NOT NULL,
  user_id       VARCHAR(64) NOT NULL,
  ad_id         VARCHAR(64) NOT NULL,
  impression_at TIMESTAMP   NOT NULL,
  click_at      TIMESTAMP
);