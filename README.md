# SPNEGO Proxy - Consul compatible

A rework of https://github.com/montag451/spnego-proxy/

## Versions

plainproxy: just a WebHDFS proxy for the namenode, doing nothing but proxying. Good to mediate latency and troubleshoot protocol.

no-consul: does SPNEGO, but doesn't use consul

consulspnegoproxy: does SPNEGO and negotiates with Consul.

(nah, i did not bother to write a consul-plain proxy, but it's a trivial matter)

## Docker

Find the last versions i bothered to build on [Docker Hub](https://hub.docker.com/r/matchalunatic/spnegoproxy/tags).
