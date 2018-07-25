package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/go-nats-streaming"
	"github.com/openfaas/faas/gateway/queue"
)

// NatsQueue queue for work
type NatsQueue struct {
	nc        stan.Conn
	ClientID  string
	ClusterID string
	NATSURL   string
	Topic     string
}

// CreateNatsQueue ready for asynchronous processing
func CreateNatsQueue(address string, port int, clientConfig NatsConfig) (*NatsQueue, error) {
	var err error
	natsURL := fmt.Sprintf("nats://%s:%d", address, port)
	log.Printf("Opening connection to %s\n", natsURL)

	clientID := clientConfig.GetClientID()
	clusterID := "faas-cluster"

	nc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	queue1 := NatsQueue{
		nc:        nc,
		ClientID:  clientID,
		ClusterID: clusterID,
		NATSURL:   natsURL,
		Topic:     "faas-request",
	}

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

	err = q.nc.Publish(q.Topic, out)

	return err
}
