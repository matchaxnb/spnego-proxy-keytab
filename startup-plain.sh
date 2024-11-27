#!/bin/sh
set -x
/spnego-proxy -addr ${LISTEN_ADDRESS} -proxy-service ${SERVICE_TO_PROXY} -debug ${APP_DEBUG}
