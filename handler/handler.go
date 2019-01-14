package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/openfaas/faas/gateway/queue"
)

// NatsQueue queue for work
type NatsQueue struct {
	nc             stan.Conn
	ncMutex        *sync.RWMutex
	maxReconnect   int
	reconnectDelay time.Duration

	ClientID  string
	ClusterID string
	NATSURL   string
	Topic     string
}

func (q *NatsQueue) connect() error {
	nc, err := stan.Connect(
		q.ClusterID,
		q.ClientID,
		stan.NatsURL(q.NATSURL),
		stan.SetConnectionLostHandler(func(conn stan.Conn, err error) {
			log.Printf("Disconnected from %s\n", q.NATSURL)

			q.reconnect()
		}),
	)

	if err != nil {
		return err
	}

	q.ncMutex.Lock()
	q.nc = nc
	q.ncMutex.Unlock()

	return nil
}

func (q *NatsQueue) reconnect() {
	for i := 0; i < q.maxReconnect; i++ {
		time.Sleep(time.Second * time.Duration(i) * q.reconnectDelay)

		if err := q.connect(); err == nil {
			log.Printf("Reconnection (%d/%d) to %s succeeded\n", i+1, q.maxReconnect, q.NATSURL)

			return
		}

		log.Printf("Reconnection (%d/%d) to %s failed\n", i+1, q.maxReconnect, q.NATSURL)
	}

	log.Printf("Reconnection limit (%d) reached\n", q.maxReconnect)
}

// CreateNatsQueue ready for asynchronous processing
func CreateNatsQueue(address string, port int, clientConfig NatsConfig) (*NatsQueue, error) {
	var err error
	natsURL := fmt.Sprintf("nats://%s:%d", address, port)
	log.Printf("Opening connection to %s\n", natsURL)

	clientID := clientConfig.GetClientID()
	clusterID := "faas-cluster"

	queue1 := NatsQueue{
		ClientID:       clientID,
		ClusterID:      clusterID,
		NATSURL:        natsURL,
		Topic:          "faas-request",
		maxReconnect:   clientConfig.GetMaxReconnect(),
		reconnectDelay: clientConfig.GetReconnectDelay(),
		ncMutex:        &sync.RWMutex{},
	}

	err = queue1.connect()

	return &queue1, err
}

// Queue request for processing
func (q *NatsQueue) Queue(req *queue.Request) error {
	fmt.Printf("NatsQueue - submitting request: %s.\n", req.Function)

	out, err := json.Marshal(req)
	if err != nil {
		log.Println(err)
	}

	q.ncMutex.RLock()
	nc := q.nc
	q.ncMutex.RUnlock()

	return nc.Publish(q.Topic, out)
}
