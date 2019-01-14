package handler

import (
	"os"
	"time"

	"github.com/openfaas/nats-queue-worker/nats"
)

type NatsConfig interface {
	GetClientID() string
	GetMaxReconnect() int
	GetReconnectDelay() time.Duration
}

type DefaultNatsConfig struct {
	maxReconnect   int
	reconnectDelay time.Duration
}

func NewDefaultNatsConfig(maxReconnect int, reconnectDelay time.Duration) DefaultNatsConfig {
	return DefaultNatsConfig{maxReconnect, reconnectDelay}
}

// GetClientID returns the ClientID assigned to this producer/consumer.
func (DefaultNatsConfig) GetClientID() string {
	val, _ := os.Hostname()
	return getClientID(val)
}

func (c DefaultNatsConfig) GetMaxReconnect() int {
	return c.maxReconnect
}

func (c DefaultNatsConfig) GetReconnectDelay() time.Duration {
	return c.reconnectDelay
}

func getClientID(hostname string) string {
	return "faas-publisher-" + nats.GetClientID(hostname)
}
