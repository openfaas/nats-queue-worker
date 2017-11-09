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
	"strings"
	"time"

	"net/http"

	"github.com/alexellis/faas/gateway/queue"
	"github.com/nats-io/go-nats-streaming"
)

// AsyncReport is the report from a function executed on a queue worker.
type AsyncReport struct {
	FunctionName string  `json:"name"`
	StatusCode   int     `json:"statusCode"`
	TimeTaken    float64 `json:"timeTaken"`
}

func printMsg(m *stan.Msg, i int) {
	log.Printf("[#%d] Received on [%s]: '%s'\n", i, m.Subject, m)
}

func makeClient() http.Client {
	proxyClient := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
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

func main() {
	log.SetFlags(0)

	clusterID := "faas-cluster"
	val, _ := os.Hostname()
	clientID := "faas-worker-" + val

	natsAddress := "nats"
	gatewayAddress := "gateway"
	functionSuffix := ""

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
	var qgroup string
	var unsubscribe bool

	client := makeClient()
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://"+natsAddress+":4222"))
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	startOpt := stan.StartWithLastReceived()
	i := 0
	mcb := func(msg *stan.Msg) {
		i++
		printMsg(msg, i)

		started := time.Now()

		req := queue.Request{}
		json.Unmarshal(msg.Data, &req)
		fmt.Printf("Request for %s.\n", req.Function)

		queryString := ""
		if len(req.QueryString) > 0 {
			queryString = fmt.Sprintf("?%s", strings.TrimLeft(req.QueryString, "?"))
		}

		functionURL := fmt.Sprintf("http://%s%s:8080/%s", req.Function, functionSuffix, queryString)

		request, err := http.NewRequest("POST", functionURL, bytes.NewReader(req.Body))
		defer request.Body.Close()

		for k, v := range req.Header {
			request.Header[k] = v
		}

		res, err := client.Do(request)
		var status int
		var functionResult []byte

		if err != nil {
			status = http.StatusServiceUnavailable

			log.Println(err)
			timeTaken := time.Since(started).Seconds()

			if req.CallbackURL != nil {
				log.Printf("Callback to: %s\n", req.CallbackURL.String())
				postResult(&client, req, functionResult, status)
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
			fmt.Println(string(functionResult))
		}
		timeTaken := time.Since(started).Seconds()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(res.Status)

		if req.CallbackURL != nil {
			log.Printf("Callback to: %s\n", req.CallbackURL.String())
			postResult(&client, req, functionResult, res.StatusCode)
		}

		postReport(&client, req.Function, res.StatusCode, timeTaken, gatewayAddress)
	}

	subj := "faas-request"
	qgroup = "faas"

	sub, err := sc.QueueSubscribe(subj, qgroup, mcb, startOpt, stan.DurableName(durable))
	if err != nil {
		log.Panicln(err)
	}

	log.Printf("Listening on [%s], clientID=[%s], qgroup=[%s] durable=[%s]\n", subj, clientID, qgroup, durable)

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

func postResult(client *http.Client, req queue.Request, result []byte, statusCode int) {
	var reader io.Reader

	if result != nil {
		reader = bytes.NewReader(result)
	}

	request, err := http.NewRequest("POST", req.CallbackURL.String(), reader)
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

	log.Printf("Posting result - %d\n", res.StatusCode)
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
	log.Printf("Posting report - %d\n", res.StatusCode)

}
