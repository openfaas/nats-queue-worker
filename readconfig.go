package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// ReadConfig constitutes config from env variables
type ReadConfig struct {
}

const DefaultMaxReconnect = 120

const DefaultReconnectDelay = time.Second * 2

func (ReadConfig) Read() (QueueWorkerConfig, error) {
	cfg := QueueWorkerConfig{
		AckWait:     time.Second * 30,
		MaxInflight: 1,
	}

	if val, exists := os.LookupEnv("faas_nats_address"); exists {
		cfg.NatsAddress = val
	} else {
		cfg.NatsAddress = "nats"
	}

	if value, exists := os.LookupEnv("faas_nats_port"); exists {
		val, err := strconv.Atoi(value)
		if err != nil {
			log.Println("converting faas_nats_port to int error:", err)
		} else {
			cfg.NatsPort = val
		}
	} else {
		cfg.NatsPort = 4222
	}

	if val, exists := os.LookupEnv("faas_nats_cluster_name"); exists {
		cfg.NatsClusterName = val
	} else {
		cfg.NatsClusterName = "faas-cluster"
	}

	if val, exists := os.LookupEnv("faas_nats_queue_group"); exists && val != "" {
		cfg.NatsQueueGroup = val
	} else {
		cfg.NatsQueueGroup = "faas"
	}

	if val, exists := os.LookupEnv("faas_gateway_address"); exists {
		cfg.GatewayAddress = val
	} else {
		cfg.GatewayAddress = "gateway"
	}

	if value, exists := os.LookupEnv("faas_gateway_port"); exists {
		val, err := strconv.Atoi(value)
		if err != nil {
			return QueueWorkerConfig{}, fmt.Errorf("converting faas_gateway_port %s to int error: %s", value, err)
		}

		cfg.GatewayPort = val

	} else {
		cfg.GatewayPort = 8080
	}

	if val, exists := os.LookupEnv("faas_print_body"); exists {
		if val == "1" || val == "true" {
			cfg.DebugPrintBody = true
		} else {
			cfg.DebugPrintBody = false
		}
	}

	if val, exists := os.LookupEnv("write_debug"); exists {
		if val == "1" || val == "true" {
			cfg.WriteDebug = true
		} else {
			cfg.WriteDebug = false
		}
	}

	cfg.MaxReconnect = DefaultMaxReconnect

	if value, exists := os.LookupEnv("faas_max_reconnect"); exists {
		val, err := strconv.Atoi(value)

		if err != nil {
			log.Println("converting faas_max_reconnect to int error:", err)
		} else {
			cfg.MaxReconnect = val
		}
	}

	cfg.ReconnectDelay = DefaultReconnectDelay

	if value, exists := os.LookupEnv("faas_reconnect_delay"); exists {
		reconnectDelayVal, durationErr := time.ParseDuration(value)

		if durationErr != nil {
			log.Println("parse env var: faas_reconnect_delay as time.Duration error:", durationErr)

		} else {
			cfg.ReconnectDelay = reconnectDelayVal
		}
	}

	if val, exists := os.LookupEnv("ack_wait"); exists {
		ackWaitVal, durationErr := time.ParseDuration(val)
		if durationErr != nil {
			log.Println("ack_wait error:", durationErr)
		} else {
			cfg.AckWait = ackWaitVal
		}
	}

	return cfg, nil
}

type QueueWorkerConfig struct {
	NatsAddress     string
	NatsPort        int
	NatsClusterName string
	NatsQueueGroup  string

	GatewayAddress string
	FunctionSuffix string
	GatewayPort    int
	MaxInflight    int
	MaxReconnect   int
	AckWait        time.Duration
	ReconnectDelay time.Duration

	DebugPrintBody bool
	WriteDebug     bool
}

func (q QueueWorkerConfig) GatewayAddressURL() string {
	return fmt.Sprintf("%s:%d", q.GatewayAddress, q.GatewayPort)
}
