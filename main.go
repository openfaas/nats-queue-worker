package main

import (
	"bytes"
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
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/openfaas/faas-provider/auth"
	"github.com/openfaas/faas/gateway/queue"
	"github.com/openfaas/nats-queue-worker/nats"
)

func main() {
	readConfig := ReadConfig{}
	config, configErr := readConfig.Read()
	if configErr != nil {
		panic(configErr)
	}

	log.SetFlags(0)

	hostname, _ := os.Hostname()

	var durable string
	if config.NatsDurableQueueSubscription {
		durable = "faas"
	}

	var unsubscribe bool
	var credentials *auth.BasicAuthCredentials
	var err error

	if config.BasicAuth {
		log.Printf("Loading basic authentication credentials")
		credentials, err = LoadCredentials()
		if err != nil {
			log.Printf("Error with LoadCredentials: %s ", err.Error())
		}
	}

	client := makeClient()

	i := 0
	messageHandler := func(msg *stan.Msg) {
		i++

		log.Printf("[#%d] Received on [%s]: '%s'\n", i, msg.Subject, msg)

		started := time.Now()

		req := queue.Request{}
		unmarshalErr := json.Unmarshal(msg.Data, &req)

		if unmarshalErr != nil {
			log.Printf("Unmarshal error: %s with data %s", unmarshalErr, msg.Data)
			return
		}

		xCallID := req.Header.Get("X-Call-Id")

		functionURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)
		fmt.Printf("Invoking: %s with %d bytes, via: %s\n", req.Function, len(req.Body), functionURL)

		if config.DebugPrintBody {
			fmt.Println(string(req.Body))
		}

		start := time.Now()
		request, err := http.NewRequest(http.MethodPost, functionURL, bytes.NewReader(req.Body))
		if err != nil {
			log.Printf("Unable to post message due to invalid URL, error: %s", err.Error())
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

		log.Printf("Invoked: %s [%d] in %fs", req.Function, statusCode, duration.Seconds())

		if err != nil {
			status = http.StatusServiceUnavailable

			log.Printf("Error invoking %s, error: %s", req.Function, err)

			timeTaken := time.Since(started).Seconds()

			if req.CallbackURL != nil {
				resultStatusCode, resultErr := postResult(&client,
					res,
					functionResult,
					req.CallbackURL.String(),
					xCallID,
					status,
					timeTaken)

				if resultErr != nil {
					log.Printf("Posted callback to: %s - status %d, error: %s\n", req.CallbackURL.String(), http.StatusServiceUnavailable, resultErr.Error())
				} else {
					log.Printf("Posted result to %s - status: %d", req.CallbackURL.String(), resultStatusCode)
				}
			}

			if config.GatewayInvoke == false {
				statusCode, reportErr := postReport(&client, req.Function, status, timeTaken, config.GatewayAddressURL(), credentials)
				if reportErr != nil {
					log.Printf("Error posting report: %s\n", reportErr)
				} else {
					log.Printf("Posting report to gateway for %s - status: %d\n", req.Function, statusCode)
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
				log.Printf("Error reading body for: %s, error: %s", req.Function, err)
			}

			if config.WriteDebug {
				fmt.Println(string(functionResult))
			} else {
				fmt.Printf("%s returned %d bytes\n", req.Function, len(functionResult))
			}
		}

		timeTaken := time.Since(started).Seconds()

		if req.CallbackURL != nil {
			log.Printf("Callback to: %s\n", req.CallbackURL.String())

			resultStatusCode, resultErr := postResult(&client,
				res,
				functionResult,
				req.CallbackURL.String(),
				xCallID,
				res.StatusCode,
				timeTaken)

			if resultErr != nil {
				log.Printf("Error posting to callback-url: %s\n", resultErr)
			} else {
				log.Printf("Posted result for %s to callback-url: %s, status: %d", req.Function, req.CallbackURL.String(), resultStatusCode)
			}
		}

		if config.GatewayInvoke == false {
			statusCode, reportErr := postReport(&client, req.Function, res.StatusCode, timeTaken, config.GatewayAddressURL(), credentials)
			if reportErr != nil {
				log.Printf("Error posting report: %s\n", reportErr.Error())
			} else {
				log.Printf("Posting report for %s, status: %d\n", req.Function, statusCode)
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
		durable:        durable,
		messageHandler: messageHandler,
		startOption:    stan.StartWithLastReceived(),
		maxInFlight:    stan.MaxInflight(config.MaxInflight),
		ackWait:        config.AckWait,
	}

	if initErr := natsQueue.connect(); initErr != nil {
		log.Panic(initErr)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
			// Do not unsubscribe a durable on exit, except if asked to.
			if durable == "" || unsubscribe {
				if err := natsQueue.unsubscribe(); err != nil {
					log.Panicf(
						"Cannot unsubscribe subject: %s from %s because of an error: %v",
						natsQueue.subject,
						natsQueue.natsURL,
						err,
					)
				}
			}
			if err := natsQueue.closeConnection(); err != nil {
				log.Panicf("Cannot close connection to %s because of an error: %v\n", natsQueue.natsURL, err)
			}
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

// makeClient constructs a HTTP client with keep-alive turned
// off and a dial-timeout of 30 seconds.
func makeClient() http.Client {
	proxyClient := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 0,
			}).DialContext,
			MaxIdleConns:          1,
			DisableKeepAlives:     true,
			IdleConnTimeout:       120 * time.Millisecond,
			ExpectContinueTimeout: 1500 * time.Millisecond,
		},
	}
	return proxyClient
}

func postResult(client *http.Client, functionRes *http.Response, result []byte, callbackURL string, xCallID string, statusCode int, timeTaken float64) (int, error) {
	var reader io.Reader

	if result != nil {
		reader = bytes.NewReader(result)
	}

	request, err := http.NewRequest(http.MethodPost, callbackURL, reader)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to post result, error: %s", err.Error())
	}

	if functionRes != nil {
		if functionRes.Header.Get("X-Duration-Seconds") == "" {
			functionRes.Header.Set("X-Duration-Seconds", fmt.Sprintf("%f", timeTaken))
		}
		copyHeaders(request.Header, &functionRes.Header)
	}

	request.Header.Set("X-Function-Status", fmt.Sprintf("%d", statusCode))

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

func postReport(client *http.Client, function string, statusCode int, timeTaken float64, gatewayAddress string, credentials *auth.BasicAuthCredentials) (int, error) {
	req := AsyncReport{
		FunctionName: function,
		StatusCode:   statusCode,
		TimeTaken:    timeTaken,
	}

	targetPostback := fmt.Sprintf("http://%s/system/async-report", gatewayAddress)
	reqBytes, _ := json.Marshal(req)
	request, err := http.NewRequest(http.MethodPost, targetPostback, bytes.NewReader(reqBytes))

	if os.Getenv("basic_auth") == "true" && credentials != nil {
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

func makeFunctionURL(req *queue.Request, config *QueueWorkerConfig, path, queryString string) string {
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
