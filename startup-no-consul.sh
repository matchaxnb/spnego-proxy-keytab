#!/bin/sh
set -x
/spnego-proxy \
  -addr "${LISTEN_ADDRESS}" \
  -config "${KRB5_CONF}" \
  -user "${KRB5_USER}" \
  -realm "${KRB5_REALM}" \
  -proxy-service "${SERVICE_TO_PROXY}" \
  -spn-service-type "${SPN_SERVICE_TYPE}" \
  -keytab-file "${KRB5_KEYTAB}" \
  -proper-username "${PROPER_USERNAME}" \
  -drop-username "${DROP_USERNAME}" \
  -metrics-addr "${METRICS_ADDRESS}" \
  -debug "${APP_DEBUG}"
