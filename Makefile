.PHONY = (all test clean plainproxydkr)
all: plain-proxy consul-proxy fixed-target-proxy

clean:
	rm -rf go.mod go.sum plain-proxy consul-proxy fixed-target-proxy
	go mod init app
	go mod tidy

plain-proxy: spnegoproxy cmd/plainproxy/*.go
	cd cmd/plainproxy; \
	go build -o ../../plain-proxy main.go

consul-proxy: spnegoproxy cmd/consulspnegoproxy/*.go
	cd cmd/consulspnegoproxy; \
	go build -o ../../consul-proxy main.go

fixed-target-proxy: spnegoproxy cmd/fixedtargetproxy/*.go
	cd cmd/fixedtargetproxy; \
	go build -o ../../fixed-target-proxy main.go

plainproxydkr:
	set -x
	docker build -t ppdkr:0.0.1 -f Dockerfile.plain .
	docker run --rm -ti --entrypoint /spnego-proxy --name plainproxy -p 50070:50070 ppdkr:0.0.1 -addr 0.0.0.0:50070 -proxy-service $(TO_PROXY)
