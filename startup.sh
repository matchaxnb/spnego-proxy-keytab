#!/bin/sh
set -x
/spnego-proxy -addr ${LISTEN_ADDRESS} -config ${KRB5_CONF} -user ${KRB5_USER} -realm ${KRB5_REALM} -consul-address ${CONSUL_ADDRESS} -proxy-service ${CONSUL_SERVICE_TO_PROXY} -spn-service-type ${SPN_SERVICE_TYPE} -keytab-file ${KRB5_KEYTAB} -debug ${APP_DEBUG}
