#!/bin/sh
set -x
/spnego-proxy -addr ${LISTEN_ADDRESS} -proxy-service ${SERVICE_TO_PROXY} -proper-username "${PROPER_USERNAME}" -debug ${APP_DEBUG}
