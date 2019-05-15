#!/bin/bash

# A simple ab-based load test

REQUESTS=${REQUESTS:-10000}
IMPRESSIONS_REQUESTS=${IMPRESSIONS_REQUESTS:-$REQUESTS}
CLICKS_REQUESTS=${CLICKS_REQUESTS:-$REQUESTS}

CONCURRENCY=${CONCURRENCY:-10}
IMPRESSIONS_CONCURRENCY=${IMPRESSIONS_CONCURRENCY:-${CONCURRENCY}}
CLICKS_CONCURRENCY=${CLICKS_CONCURRENCY:-${CONCURRENCY}}

REGION=${REGION:-${AWS_DEFAULT_REGION}}
STAGE=${STAGE:-Prod}

BASE_URL="https://${API_ID}.execute-api.${REGION}.amazonaws.com/${STAGE}"

ab -k -n ${IMPRESSIONS_REQUESTS} -c ${IMPRESSIONS_CONCURRENCY} -H 'Content-Type: application/json' -p ./example-impressions.json "${BASE_URL}/data/ad/impressions" &
ab -k -n ${CLICKS_REQUESTS} -c ${CLICKS_CONCURRENCY} -H 'Content-Type: application/json' -p ./example-clicks.json "${BASE_URL}/data/ad/impressions" &

wait
