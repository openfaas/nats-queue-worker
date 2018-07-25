package handler

import (
	"encoding/json"
	"fmt"
	"log"

	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/openfaas/faas/gateway/queue"
)

// NatsQueue queue for work
type NatsQueue struct {
	stanConn  stan.Conn
	natsConn  *nats.Conn
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

	queue1 := NatsQueue{
		ClientID:  clientID,
		ClusterID: clusterID,
		NATSURL:   natsURL,
		Topic:     "faas-request",
	}

	natsConn, err := nats.Connect(natsURL, nats.ReconnectHandler(queue1.reconnectClient(clientID, clusterID, natsURL)))
	if err != nil {
		return nil, err
	}

	stanConn, err := stan.Connect(clusterID, clientID, stan.NatsConn(natsConn))
	if err != nil {
		return nil, err
	}

	queue1.natsConn = natsConn
	queue1.stanConn = stanConn

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

	err = q.stanConn.Publish(q.Topic, out)

	return err
}

func (q *NatsQueue) reconnectClient(clientID, clusterID, natsURL string) nats.ConnHandler {
	return func(c *nats.Conn) {
		oldConn := q.stanConn

		defer oldConn.Close()

		stanConn, err := stan.Connect(clusterID, clientID, stan.NatsConn(c))
		if err != nil {
			log.Printf("Failed to reconnect to NATS\n%v", err)
			return
		} else {
			log.Printf("Reconnected to NATS\n")
		}

		q.stanConn = stanConn
		q.natsConn = c
	}
}
