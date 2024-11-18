package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	capi "github.com/hashicorp/consul/api"

	"github.com/matchaxnb/gokrb5/v8/client"
	"github.com/matchaxnb/gokrb5/v8/config"
	"github.com/matchaxnb/gokrb5/v8/keytab"
	"github.com/matchaxnb/gokrb5/v8/spnego"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

const MAX_ERROR_COUNT = 20
const PAUSE_TIME_WHEN_ERROR = time.Minute * 1

type SPNEGOClient struct {
	Client *spnego.SPNEGO
	mu     sync.Mutex
}

func (c *SPNEGOClient) GetToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.Client.AcquireCred(); err != nil {
		return "", fmt.Errorf("could not acquire client credential: %v", err)
	}
	token, err := c.Client.InitSecContext()
	if err != nil {
		return "", fmt.Errorf("could not initialize context: %v", err)
	}
	b, err := token.Marshal()
	if err != nil {
		return "", fmt.Errorf("could not marshal SPNEGO token: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func startConsulGetService(client *capi.Client, serviceName string) chan []HostPort {
	messages := make(chan []HostPort)
	serviceFunc := func(client *capi.Client, serviceName string, messages chan []HostPort) {
		healthyServices, meta, err := client.Health().Service(serviceName, "", true, &capi.QueryOptions{})
		if err != nil {
			logger.Printf("Cannot get healthy services for %#v (response meta: %#v) because of a consul error: %s", serviceName, meta, err)
			return
		}
		healthyStrings := make([]HostPort, len(healthyServices))
		for i := range healthyServices {
			log.Printf("Service: %#v\n", healthyServices[i].Node.Meta["fqdn"])
			healthyStrings[i] = HostPort{healthyServices[i].Node.Meta["fqdn"], healthyServices[i].Service.Port}
		}
		messages <- healthyStrings
		time.Sleep(time.Second * 30)
	}
	go serviceFunc(client, serviceName, messages)
	return messages
}

type HostPort struct {
	Host string
	Port int
}

func (e HostPort) f() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func buildSPNClient(validHosts chan []HostPort, krbClient *client.Client, serviceType string) (spnClient *SPNEGOClient, realSpn string, realHost string, err error) {
	// pick the first valid host from our chan
	logger.Print("Building a spn client")
	spnHost := <-validHosts
	spnStr := fmt.Sprintf("%s/%s", serviceType, spnHost[0].Host)
	return &SPNEGOClient{
		Client: spnego.SPNEGOClient(krbClient, spnStr),
	}, spnStr, spnHost[0].f(), nil
}

func main() {
	addr := flag.String("addr", "0.0.0.0:50070", "bind address")
	cfgFile := flag.String("config", "krb5.conf", "krb5 config file")
	user := flag.String("user", "your.user/your.host", "user name")
	realm := flag.String("realm", "YOUR.REALM", "realm")
	consulAddress := flag.String("consul-address", "your.consul.host:8500", "consul server address")
	consulToken := flag.String("consul-token", "", "consul access token (optional)")
	proxy := flag.String("proxy-service", "your-service-to-proxy", "proxy consul service")
	spnServiceType := flag.String("spn-service-type", "HTTP", "SPN service type")
	keytabFile := flag.String("keytab-file", "krb5.keytab", "keytab file path")
	debug := flag.Bool("debug", true, "turn on debugging")
	flag.Parse()
	keytab, err := keytab.Load(*keytabFile)
	if err != nil {
		logger.Printf("cannot read keytab: %s\n", err)
		logger.Panic("no keytab no dice")
	}
	conf, err := config.Load(*cfgFile)
	unsupErr := config.UnsupportedDirective{}
	if err != nil && !errors.As(err, &unsupErr) {
		logger.Printf("Bad config: %s\n", err)
		logger.Panic("no config no dice")
	}

	consulClient, err := capi.NewClient(&capi.Config{Address: *consulAddress, Scheme: "http", Token: *consulToken})
	if err != nil {
		logger.Panicf("Cannot connect to consul: %s", err)
	}
	realHosts := startConsulGetService(consulClient, *proxy)
	kclient := client.NewWithKeytab(*user, *realm, keytab, conf, client.Logger(logger), client.DisablePAFXFAST(false))
	kclient.Login()
	spnegoClient, spnEnabled, realHost, err := buildSPNClient(realHosts, kclient, *spnServiceType)
	if err != nil {
		logger.Panic("Cannot get SPN for service, failing")
	}
	_, _, err = kclient.GetServiceTicket(spnEnabled)
	if err != nil {
		log.Panic("Cannot get service ticket, probably wrong config", err)
	}
	if *debug {
		logger.Printf("Listening on %s\n", *addr)
	}
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Panic(err)
	}
	errorCount := 0
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Panic(err)
		}
		go handleClient(conn, realHost, spnegoClient, *debug, &errorCount)
	}
}

func handleClient(conn net.Conn, proxyHost string, spnegoCli *SPNEGOClient, debug bool, errCount *int) {
	if *errCount > MAX_ERROR_COUNT {
		log.Fatalf("Too many errors (%d), exiting", *errCount)
	}
	defer conn.Close()
	if debug {
		defer logger.Printf("stop processing request for client: %v", conn.RemoteAddr())
		logger.Printf("new client: %v", conn.RemoteAddr())
	}
	proxyConn, err := net.Dial("tcp", proxyHost)
	if err != nil {
		logger.Panicf("failed to connect to proxy: %v", err)
		return
	}
	defer proxyConn.Close()
	reqReader := bufio.NewReader(conn)
	if debug {
		reqReader = bufio.NewReader(io.TeeReader(conn, os.Stdout))
	}
	token, err := spnegoCli.GetToken()
	if err != nil {
		logger.Printf("failed to get SPNEGO token: %v", err)
		time.Sleep(PAUSE_TIME_WHEN_ERROR)
		*errCount += 1
		return
	}
	authHeader := "Negotiate " + token
	req, err := http.ReadRequest(reqReader)
	if err != nil {
		*errCount += 1
		if !errors.Is(err, io.EOF) {
			logger.Printf("failed to read request: %v", err)
		}
		return
	}
	req.Host = proxyHost
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Connection", "close")
	req.Header.Set("User-agent", "hadoop-proxy/0.1")

	if debug {
		req.WriteProxy(io.MultiWriter(proxyConn, os.Stdout))
	} else {
		req.WriteProxy(proxyConn)
	}
	var wg sync.WaitGroup
	forward := func(from, to net.Conn, tag string) {
		defer wg.Done()
		defer to.(*net.TCPConn).CloseWrite()
		if debug {
			fromAddr, toAddr := from.RemoteAddr(), to.RemoteAddr()
			logger.Printf("[%s] forward start %v -> %v", tag, fromAddr, toAddr)
			defer logger.Printf("[%s] forward done %v -> %v", tag, fromAddr, toAddr)
		}
		io.Copy(to, from)
	}
	wg.Add(2)
	go forward(conn, proxyConn, "local to proxied")
	go forward(proxyConn, conn, "proxied to local")
	wg.Wait()
}
