package handler

import (
	"os"

	"github.com/openfaas/nats-queue-worker/nats"
)

type NatsConfig interface {
	GetClientID() string
}

type DefaultNatsConfig struct {
}

// GetClientID returns the ClientID assigned to this producer/consumer.
func (DefaultNatsConfig) GetClientID() string {
	val, _ := os.Hostname()
	return getClientID(val)
}

func getClientID(hostname string) string {
	return "faas-publisher-" + nats.GetClientID(hostname)
}
