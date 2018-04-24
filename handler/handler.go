package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/openfaas/faas/gateway/queue"
)

// NatsQueue queue for work
type NatsQueue struct {
	nc       stan.Conn
	natsConn *nats.Conn
}

// CreateNatsQueue ready for asynchronous processing
func CreateNatsQueue(address string, port int) (*NatsQueue, error) {
	queue1 := NatsQueue{}
	var err error
	natsURL := fmt.Sprintf("nats://%s:%d", address, port)
	log.Printf("Opening connection to %s\n", natsURL)

	val, _ := os.Hostname()
	clientID := "faas-publisher-" + val
	clusterID := "faas-cluster"

	natsConn, err := nats.Connect(natsURL, nats.ReconnectHandler(queue1.reconnectClient(clientID, clusterID, natsURL)))
	if err != nil {
		return nil, err
	}

	nc, err := stan.Connect(clusterID, clientID, stan.NatsConn(natsConn))
	if err != nil {
		return nil, err
	}
	queue1.nc = nc
	queue1.natsConn = natsConn

	return &queue1, err
}

// Queue request for processing
func (q *NatsQueue) Queue(req *queue.Request) error {
	var err error

	fmt.Printf("NatsQueue - submitting request: %s.\n", req.Function)

	out, err := json.Marshal(req)
	if err != nil {
		log.Println(err)
	}

	err = q.nc.Publish("faas-request", out)

	return err
}

func (q *NatsQueue) reconnectClient(clientID, clusterID, natsURL string) nats.ConnHandler {
	return func(c *nats.Conn) {
		q.nc.Close()
		nc, err := stan.Connect(clusterID, clientID, stan.NatsConn(c))
		if err != nil {
			log.Printf("Failed to reconnect to NATS stream\n%v", err)
			return
		}
		q.nc = nc
		q.natsConn = c
	}
}
