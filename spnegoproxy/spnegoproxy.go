package spnegoproxy

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	capi "github.com/hashicorp/consul/api"

	"github.com/matchaxnb/gokrb5/v8/client"
	"github.com/matchaxnb/gokrb5/v8/config"
	"github.com/matchaxnb/gokrb5/v8/keytab"
	"github.com/matchaxnb/gokrb5/v8/spnego"
)

var logger = log.New(os.Stderr, "[spnegoproxy]", log.LstdFlags)

const MAX_ERROR_COUNT = 20
const PAUSE_TIME_WHEN_ERROR = time.Minute * 1
const PAUSE_TIME_WHEN_NO_DATA = time.Millisecond * 300

type SPNEGOClient struct {
	Client *spnego.SPNEGO
	mu     sync.Mutex
}

type HostPort struct {
	Host string
	Port int
}

func (e HostPort) f() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func BuildConsulClient(consulAddress *string, consulToken *string) *capi.Client {
	consulClient, err := capi.NewClient(&capi.Config{Address: *consulAddress, Scheme: "http", Token: *consulToken})
	if err != nil {
		logger.Panicf("Cannot connect to consul: %s", err)
	}
	return consulClient
}

func BuildSPNClient(validHosts chan []HostPort, krbClient *client.Client, serviceType string) (spnClient *SPNEGOClient, realSpn string, realHost string, err error) {
	// pick the first valid host from our chan
	logger.Print("Building a spn client")
	spnHost := <-validHosts
	spnStr := fmt.Sprintf("%s/%s", serviceType, spnHost[0].Host)
	return &SPNEGOClient{
		Client: spnego.SPNEGOClient(krbClient, spnStr),
	}, spnStr, spnHost[0].f(), nil
}

func LoadKrb5Config(keytabFile *string, cfgFile *string) (*keytab.Keytab, *config.Config) {
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
	return keytab, conf
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

func HostnameToChanHostPort(hostname string) chan []HostPort {
	messages := make(chan []HostPort)
	spl := strings.Split(hostname, ":")
	if len(spl) != 2 {
		logger.Panicf("Could not split %s by character : and get 2 bits", hostname)
	}
	portNum, err := strconv.Atoi(spl[1])
	if err != nil {
		logger.Panicf("Cannot parse %s to int: %s", spl[1], err)
	}
	hp := HostPort{spl[0], portNum}
	messages <- []HostPort{hp}
	return messages
}

func StartConsulGetService(client *capi.Client, serviceName string) chan []HostPort {
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

// func HandleClientWithoutSPNEGO

func HandleClient(conn *net.TCPConn, proxyHost string, spnegoCli *SPNEGOClient, debug bool, errCount *int) {
	if *errCount > MAX_ERROR_COUNT {
		log.Fatalf("Too many errors (%d), exiting", *errCount)
	}
	if debug {
		logger.Printf("new client: %v", conn.RemoteAddr())
		defer logger.Printf("stop processing request for client: %v", conn.RemoteAddr())

	}
	defer conn.Close()
	proxyAddr, err := net.ResolveTCPAddr("tcp", proxyHost)
	if err != nil {
		logger.Panicf("Cannot resolve proxy hostname %s -> %s", proxyHost, err)
	}

	proxyConn, err := net.DialTCP("tcp", nil, proxyAddr)
	if err != nil {
		logger.Panicf("failed to connect to proxy: %v", err)
		return
	}
	defer proxyConn.Close()
	reqReader := bufio.NewReader(conn)

	/*if debug {
		reqReader = bufio.NewReader(io.TeeReader(conn, os.Stdout))
	}*/

	// get the SPNEGO token that we will use for this client

	if debug {
		if spnegoCli == nil {
			logger.Print("no SPNEGO client is set, so no Kerberos auth happening (this is fine)")
		}
	}
	processedCounter := 0
	var wg sync.WaitGroup
	for {
		req, err := readRequestAndSetAuthorization(reqReader, spnegoCli)
		if err != nil && !errors.Is(err, io.EOF) {
			logger.Printf("failed to read request or to get SPNEGO token: %v", err)
			*errCount += 1
			time.Sleep(PAUSE_TIME_WHEN_NO_DATA)
			continue
		}
		if errors.Is(err, net.ErrClosed) {
			logger.Print("HandleClient: socket closed")
			break
		}
		logger.Printf("Read request: %s", req.URL)
		req.Host = proxyHost
		req.Header.Set("User-agent", "hadoop-proxy/0.1")
		req.WriteProxy(proxyConn)

		forward := func(from, to *net.TCPConn, tag string, isResponse bool) {
			defer wg.Done()
			// defer to.CloseWrite()
			fromAddr, toAddr := from.RemoteAddr(), to.RemoteAddr()
			if !isResponse {
				logger.Printf("[%s] request %s -> %s\n", tag, fromAddr, toAddr)
				io.Copy(to, from) // this is optimized but removes control
			} else {
				logger.Printf("[%s] response %s -> %s\n", tag, fromAddr, toAddr)
				// read the from
				resReader := bufio.NewReader(from)

				res, err := http.ReadResponse(resReader, nil)
				if err != nil {
					logger.Panicf("[%s] Could not read response: %s", tag, err)
				}
				//res.Header.Del("Www-Authenticate")
				//res.Header.Del("Setb-Cookie")
				res.Write(to)
			}
			logger.Printf("[%s] written\n", tag)
			//from.CloseRead()
			to.CloseWrite()

		}
		wg.Add(2)
		go forward(conn, proxyConn, "local to proxied", false)
		go forward(proxyConn, conn, "proxied to local", true)

		*errCount = 0
		processedCounter += 1
	}
	wg.Wait()
	logger.Printf("[ProcessedCounter] Handled %d requests\n", processedCounter)
}

func readRequestAndSetAuthorization(reqReader *bufio.Reader, spnegoCli *SPNEGOClient) (*http.Request, error) {
	authHeader := ""
	req, err := http.ReadRequest(reqReader)
	if err != nil {
		return nil, err
	}
	if spnegoCli != nil {
		token, err := spnegoCli.GetToken()
		if err != nil {
			logger.Printf("failed to get SPNEGO token: %v", err)
			time.Sleep(PAUSE_TIME_WHEN_ERROR)

			return nil, err
		}
		authHeader = "Negotiate " + token
	}

	if len(authHeader) > 0 {
		req.Header.Set("Authorization", authHeader)
	}
	return req, nil
}
