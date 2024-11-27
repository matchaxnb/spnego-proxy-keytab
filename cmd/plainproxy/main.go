package main

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/matchaxnb/spnegoproxy/spnegoproxy"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	addr := flag.String("addr", "0.0.0.0:50070", "bind address")
	toProxy := flag.String("proxy-service", "", "host:port for the service to proxy to")
	properUsername := flag.String("proper-username", "", "for WebHDFS, user.name value to force-set")
	debug := flag.Bool("debug", true, "turn on debugging")
	flag.Parse()
	if *debug {
		logger.Printf("Listening on %s\n", *addr)
	}
	if len(*toProxy) == 0 {
		logger.Fatal("Need to provide -proxy-service flag")
	}
	listenAddr, err := net.ResolveTCPAddr("tcp", *addr)
	if err != nil {
		logger.Panicf("Wrong TCP address %s -> %s", *addr, err)
	}
	connListener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		logger.Panic(err)
	}
	errorCount := 0
	defer connListener.Close()
	for {
		conn, err := connListener.AcceptTCP()
		if err != nil {
			logger.Panic(err)
		}
		go spnegoproxy.HandleClient(conn, *toProxy, nil, *properUsername, *debug, &errorCount)
	}
}
