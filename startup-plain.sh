#!/bin/sh
set -x
/spnego-proxy \
  -addr ${LISTEN_ADDRESS} \
  -proxy-service ${SERVICE_TO_PROXY} \
  -proper-username "${PROPER_USERNAME}" \
  -metrics-addr "${METRICS_ADDRESS}" \
  -debug "${APP_DEBUG}"
