FROM golang:latest
WORKDIR /src
COPY *.go go.mod go.sum .
RUN go mod tidy
RUN go build -o /spnego-proxy -ldflags '-linkmode external -extldflags "-fno-PIC -static"'
FROM alpine:latest
WORKDIR /data
COPY --from=0 /spnego-proxy /spnego-proxy
ENV LISTEN_ADDRESS="0.0.0.0:50070" KRB5_CONF="/data/krb5.conf" \
    KRB5_KEYTAB="/data/krb5.keytab" KRB5_REALM="YOUR.REALM" \
    KRB5_USER="youruser/your.host" \
    CONSUL_ADDRESS="your.consul.address" \
    CONSUL_SERVICE_TO_PROXY="your-consul-service" \
    SPN_SERVICE_TYPE="HTTP" APP_DEBUG="false"
SHELL [ "/bin/sh", "-c"]
EXPOSE 50070
COPY startup.sh /startup.sh
ENTRYPOINT [ "/startup.sh"]
