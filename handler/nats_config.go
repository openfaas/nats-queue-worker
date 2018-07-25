package handler

import (
	"os"
	"regexp"
)

type NatsConfig interface {
	GetClientID() string
}

type DefaultNatsConfig struct {
}

var supportedCharacters, _ = regexp.Compile("[^a-zA-Z0-9-_]+")

// GetClientID returns the ClientID assigned to this producer/consumer.
func (DefaultNatsConfig) GetClientID() string {
	val, _ := os.Hostname()
	return getClientID(val)
}

func getClientID(hostname string) string {
	return "faas-publisher-" + supportedCharacters.ReplaceAllString(hostname, "_")
}
