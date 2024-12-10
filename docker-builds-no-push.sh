#!/bin/sh
WANTED_TAG=$1

docker build -f Dockerfile -t matchalunatic/spnegoproxy:consul-${WANTED_TAG} .
docker build -f Dockerfile.no-consul -t matchalunatic/spnegoproxy:fixedtarget-${WANTED_TAG} .
docker build -f Dockerfile.plain -t matchalunatic/spnegoproxy:plain-${WANTED_TAG} .
