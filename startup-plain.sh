#!/bin/sh
set -x
/spnego-proxy \
  -addr "${LISTEN_ADDRESS}" \
  -metrics-addr "${METRICS_ADDRESS}" \
  -proxy-service "${SERVICE_TO_PROXY}" \
  -proper-username "${PROPER_USERNAME}" \
  -drop-username "${DROP_USERNAME}" \
  -debug "${APP_DEBUG}"
