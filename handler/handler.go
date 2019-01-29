package handler

import (
	"fmt"
	"log"
	"sync"
)

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
