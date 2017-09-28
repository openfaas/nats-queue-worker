package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"net/http"

	"github.com/alexellis/faas/gateway/queue"
	"github.com/nats-io/go-nats-streaming"
)

func main() {
	log.SetFlags(0)

	clusterID := "faas-cluster"
	val, _ := os.Hostname()
	clientID := "faas-worker-" + val

	natsAddress := "nats"
	gatewayAddress := "gateway"
	functionSuffix := ""
	faasWriteDebug := true

	if val, exists := os.LookupEnv("faas_write_debug"); exists {
		faasWriteDebug = (val == "false")
	}

	if val, exists := os.LookupEnv("faas_nats_address"); exists {
		natsAddress = val
	}

	if val, exists := os.LookupEnv("faas_gateway_address"); exists {
		gatewayAddress = val
	}

	if val, exists := os.LookupEnv("faas_function_suffix"); exists {
		functionSuffix = val
	}

	var durable string
	var queueGroup string
	var unsubscribe bool

	dialTimeout := 3 * time.Second

	// Same client instance is reused.
	client := makeClient(dialTimeout)
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://"+natsAddress+":4222"))
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	startOpt := stan.StartWithLastReceived()
	i := 0
	mcb := func(msg *stan.Msg) {
		i++

		printMsg(msg, i, faasWriteDebug)

		started := time.Now()

		req := queue.Request{}
		json.Unmarshal(msg.Data, &req)
		fmt.Printf("Request for %s.\n", req.Function)

		// POSTs directly to function via DNS lookup using req.Function name.
		urlFunction := fmt.Sprintf("http://%s%s:8080/", req.Function, functionSuffix)

		request, err := http.NewRequest("POST", urlFunction, bytes.NewReader(req.Body))
		defer request.Body.Close()

		res, err := client.Do(request)
		var status int
		var functionResult []byte

		if err != nil {
			status = http.StatusServiceUnavailable

			log.Println(err)
			timeTaken := time.Since(started).Seconds()

			if req.CallbackURL != nil {
				log.Printf("Callback to: %s\n", req.CallbackURL.String())
				resultReader := bytes.NewReader(functionResult)
				postResult(&client, req, resultReader, status)
			}

			postReport(&client, req.Function, status, timeTaken, gatewayAddress)
			return
		}

		if res.Body != nil {
			defer res.Body.Close()

			resData, err := ioutil.ReadAll(res.Body)
			functionResult = resData

			if err != nil {
				log.Println(err)
			}
			if faasWriteDebug {
				fmt.Println(string(functionResult))
			}
		}

		timeTaken := time.Since(started).Seconds()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("Result: %s\n", res.Status)

		if req.CallbackURL != nil {
			log.Printf("Callback to: %s\n", req.CallbackURL.String())
			resultReader := bytes.NewReader(functionResult)
			postResult(&client, req, resultReader, res.StatusCode)
		}

		postReport(&client, req.Function, res.StatusCode, timeTaken, gatewayAddress)
	}

	subj := "faas-request"
	queueGroup = "faas"

	sub, err := sc.QueueSubscribe(subj, queueGroup, mcb, startOpt, stan.DurableName(durable))
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("Listening on [%s], clientID=[%s], qgroup=[%s] durable=[%s]\n", subj, clientID, queueGroup, durable)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")

			// Do not unsubscribe a durable on exit, except if asked to.
			if durable == "" || unsubscribe {
				sub.Unsubscribe()
			}

			sc.Close()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

func printMsg(m *stan.Msg, i int, writeDebug bool) {
	if writeDebug {
		log.Printf("[#%d] Received on [%s]: '%s'\n", i, m.Subject, m)
	} else {
		log.Printf("[#%d] Received on [%s].\n", i, m.Subject)
	}
}

func makeClient(dialTimeout time.Duration) http.Client {
	proxyClient := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 0,
			}).DialContext,
			MaxIdleConns:          1,
			DisableKeepAlives:     true, // enables round-robin behavior
			IdleConnTimeout:       120 * time.Millisecond,
			ExpectContinueTimeout: 1500 * time.Millisecond,
		},
	}
	return proxyClient
}

// postResult - only POST method is supported for calling back with result.
func postResult(client *http.Client, req queue.Request, reader io.Reader, statusCode int) {

	callbackURL := req.CallbackURL.String()

	request, err := http.NewRequest("POST", callbackURL, reader)
	res, err := client.Do(request)

	if err != nil {
		log.Printf("Error posting result to URL %s %s\n", req.CallbackURL.String(), err.Error())
		return
	}

	if request.Body != nil {
		defer request.Body.Close()
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	log.Printf("Posting result [%d] to: %s\n", res.StatusCode, callbackURL)
}

func postReport(client *http.Client, function string, statusCode int, timeTaken float64, gatewayAddress string) {
	req := AsyncReport{
		FunctionName: function,
		StatusCode:   statusCode,
		TimeTaken:    timeTaken,
	}

	reqBytes, _ := json.Marshal(req)
	request, err := http.NewRequest("POST", "http://"+gatewayAddress+":8080/system/async-report", bytes.NewReader(reqBytes))
	defer request.Body.Close()

	res, err := client.Do(request)

	if err != nil {
		log.Println("Error posting report", err)
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	log.Printf("Posting report to gateway: %d\n", res.StatusCode)
}
