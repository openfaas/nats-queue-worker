package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	stan "github.com/nats-io/stan.go"

	"github.com/openfaas/faas-provider/auth"
	ftypes "github.com/openfaas/faas-provider/types"
	"github.com/openfaas/nats-queue-worker/nats"
	"github.com/openfaas/nats-queue-worker/version"
)

func main() {
	readConfig := ReadConfig{}
	config, configErr := readConfig.Read()
	if configErr != nil {
		panic(configErr)
	}

	log.SetFlags(0)

	hostname, _ := os.Hostname()

	var credentials *auth.BasicAuthCredentials
	var err error

	if config.BasicAuth {
		log.Printf("Loading basic authentication credentials")
		credentials, err = LoadCredentials()
		if err != nil {
			log.Printf("Error with LoadCredentials: %s ", err.Error())
		}
	}

	sha, release := version.GetReleaseInfo()
	log.Printf("Starting queue-worker. Version: %s\tGit Commit: %s", release, sha)

	client := makeClient(config.TLSInsecure)

	counter := uint64(0)
	messageHandler := func(msg *stan.Msg) {
		i := atomic.AddUint64(&counter, 1)

		log.Printf("[#%d] Received on [%s]: '%s'\n", i, msg.Subject, msg)

		started := time.Now()

		req := ftypes.QueueRequest{}
		unmarshalErr := json.Unmarshal(msg.Data, &req)

		if unmarshalErr != nil {
			log.Printf("[#%d] Unmarshal error: %s with data %s", i, unmarshalErr, msg.Data)
			return
		}

		xCallID := req.Header.Get("X-Call-Id")

		functionURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)
		fmt.Printf("[#%d] Invoking: %s with %d bytes, via: %s\n", i, req.Function, len(req.Body), functionURL)

		if config.DebugPrintBody {
			fmt.Println(string(req.Body))
		}

		start := time.Now()
		request, err := http.NewRequest(http.MethodPost, functionURL, bytes.NewReader(req.Body))
		if err != nil {
			log.Printf("[#%d] Unable to post message due to invalid URL, error: %s", i, err.Error())
			return
		}

		defer request.Body.Close()
		copyHeaders(request.Header, &req.Header)

		res, err := client.Do(request)

		var status int
		var functionResult []byte

		var statusCode int
		if err != nil {
			statusCode = http.StatusServiceUnavailable
		} else {
			statusCode = res.StatusCode
		}

		duration := time.Since(start)

		log.Printf("[#%d] Invoked: %s [%d] in %fs", i, req.Function, statusCode, duration.Seconds())

		if err != nil {
			status = http.StatusServiceUnavailable

			log.Printf("[#%d] Error invoking %s, error: %s", i, req.Function, err)

			timeTaken := time.Since(started).Seconds()

			if req.CallbackURL != nil {
				resultStatusCode, resultErr := postResult(&client,
					res,
					functionResult,
					req.CallbackURL.String(),
					xCallID,
					status,
					req.Function,
					timeTaken)

				if resultErr != nil {
					log.Printf("[#%d] Posted callback to: %s - status %d, error: %s\n", i, req.CallbackURL.String(), http.StatusServiceUnavailable, resultErr.Error())
				} else {
					log.Printf("[#%d] Posted result to %s - status: %d", i, req.CallbackURL.String(), resultStatusCode)
				}
			}

			if config.GatewayInvoke == false {
				statusCode, reportErr := postReport(&client, req.Function, status, timeTaken, &config, credentials)
				if reportErr != nil {
					log.Printf("[#%d] Error posting report: %s\n", i, reportErr)
				} else {
					log.Printf("[#%d] Posting report to gateway for %s - status: %d\n", i, req.Function, statusCode)
				}
				return
			}

			return
		}

		if res.Body != nil {
			defer res.Body.Close()

			resData, err := ioutil.ReadAll(res.Body)
			functionResult = resData

			if err != nil {
				log.Printf("[#%d] Error reading body for: %s, error: %s", i, req.Function, err)
			}

			if config.WriteDebug {
				fmt.Println(string(functionResult))
			} else {
				fmt.Printf("[#%d] %s returned %d bytes\n", i, req.Function, len(functionResult))
			}
		}

		timeTaken := time.Since(started).Seconds()

		if req.CallbackURL != nil {
			log.Printf("[#%d] Callback to: %s\n", i, req.CallbackURL.String())

			resultStatusCode, resultErr := postResult(&client,
				res,
				functionResult,
				req.CallbackURL.String(),
				xCallID,
				res.StatusCode,
				req.Function,
				timeTaken)

			if resultErr != nil {
				log.Printf("[#%d] Error posting to callback-url: %s\n", i, resultErr)
			} else {
				log.Printf("[#%d] Posted result for %s to callback-url: %s, status: %d", i, req.Function, req.CallbackURL.String(), resultStatusCode)
			}
		}

		if config.GatewayInvoke == false {
			statusCode, reportErr := postReport(&client, req.Function, res.StatusCode, timeTaken, &config, credentials)
			if reportErr != nil {
				log.Printf("[#%d] Error posting report: %s\n", i, reportErr.Error())
			} else {
				log.Printf("[#%d] Posting report for %s, status: %d\n", i, req.Function, statusCode)
			}
		}

	}

	natsURL := fmt.Sprintf("nats://%s:%d", config.NatsAddress, config.NatsPort)

	natsQueue := NATSQueue{
		clusterID: config.NatsClusterName,
		clientID:  "faas-worker-" + nats.GetClientID(hostname),
		natsURL:   natsURL,

		connMutex:      &sync.RWMutex{},
		maxReconnect:   config.MaxReconnect,
		reconnectDelay: config.ReconnectDelay,
		quitCh:         make(chan struct{}),

		subject:        config.NatsChannel,
		qgroup:         config.NatsQueueGroup,
		messageHandler: messageHandler,
		maxInFlight:    config.MaxInflight,
		ackWait:        config.AckWait,
	}

	if initErr := natsQueue.connect(); initErr != nil {
		log.Panic(initErr)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
	if err := natsQueue.closeConnection(); err != nil {
		log.Panicf("Cannot close connection to %s because of an error: %v\n", natsQueue.natsURL, err)
	}
	close(signalChan)
}

// makeClient constructs a HTTP client with keep-alive turned
// off and a dial-timeout of 30 seconds.
//
// tlsInsecure is required for callbacks to internal services which
// may not have a trusted CA root, or may be misconfigured
func makeClient(tlsInsecure bool) http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 0,
		}).DialContext,

		MaxIdleConns:          1,
		DisableKeepAlives:     true,
		IdleConnTimeout:       120 * time.Millisecond,
		ExpectContinueTimeout: 1500 * time.Millisecond,
	}
	if tlsInsecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: tlsInsecure}
	}

	proxyClient := http.Client{
		Transport: tr,
	}

	return proxyClient
}

func postResult(client *http.Client, functionRes *http.Response, result []byte, callbackURL string, xCallID string,
	statusCode int, functionName string, timeTaken float64) (int, error) {
	var reader io.Reader

	if result != nil {
		reader = bytes.NewReader(result)
	}

	request, err := http.NewRequest(http.MethodPost, callbackURL, reader)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to post result, error: %s", err.Error())
	}

	if functionRes != nil {
		copyHeaders(request.Header, &functionRes.Header)
	}

	request.Header.Set("X-Duration-Seconds", fmt.Sprintf("%f", timeTaken))
	request.Header.Set("X-Function-Status", fmt.Sprintf("%d", statusCode))
	request.Header.Set("X-Function-Name", functionName)

	if len(xCallID) > 0 {
		request.Header.Set("X-Call-Id", xCallID)
	}

	res, err := client.Do(request)

	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error posting result to URL %s %s", callbackURL, err.Error())
	}

	if request.Body != nil {
		defer request.Body.Close()
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	return res.StatusCode, nil
}

func copyHeaders(destination http.Header, source *http.Header) {
	for k, v := range *source {
		vClone := make([]string, len(v))
		copy(vClone, v)
		(destination)[k] = vClone
	}
}

func postReport(client *http.Client, function string, statusCode int, timeTaken float64, config *QueueWorkerConfig, credentials *auth.BasicAuthCredentials) (int, error) {
	req := AsyncReport{
		FunctionName: function,
		StatusCode:   statusCode,
		TimeTaken:    timeTaken,
	}

	targetPostback := fmt.Sprintf("http://%s/system/async-report", config.GatewayAddressURL())
	reqBytes, _ := json.Marshal(req)
	request, err := http.NewRequest(http.MethodPost, targetPostback, bytes.NewReader(reqBytes))

	if config.BasicAuth && credentials != nil {
		request.SetBasicAuth(credentials.User, credentials.Password)
	}

	defer request.Body.Close()

	res, err := client.Do(request)

	if err != nil {
		return http.StatusGatewayTimeout, fmt.Errorf("cannot post report to %s: %s", targetPostback, err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	return res.StatusCode, nil
}

func makeFunctionURL(req *ftypes.QueueRequest, config *QueueWorkerConfig, path, queryString string) string {
	qs := ""
	if len(queryString) > 0 {
		qs = fmt.Sprintf("?%s", strings.TrimLeft(queryString, "?"))
	}
	pathVal := "/"
	if len(path) > 0 {
		pathVal = path
	}

	var functionURL string
	if config.GatewayInvoke {
		functionURL = fmt.Sprintf("http://%s/function/%s%s%s",
			config.GatewayAddressURL(),
			strings.Trim(req.Function, "/"),
			pathVal,
			qs)
	} else {
		functionURL = fmt.Sprintf("http://%s%s:8080%s%s",
			req.Function,
			config.FunctionSuffix,
			pathVal,
			qs)
	}

	return functionURL
}
