## Notice

[NATS Streaming](https://github.com/nats-io/nats-streaming-server) was deprecated in June 2023 by Synadia, and will receive no more updates, including for critical security issues.

Migrate to OpenFaaS Standard for NATS JetStream, learn more:

* [Docs: JetStream for OpenFaaS](https://docs.openfaas.com/openfaas-pro/jetstream/)
* [Announcement: The Next Generation of Queuing: JetStream for OpenFaaS](https://www.openfaas.com/blog/jetstream-for-openfaas/)

## queue-worker (Community Edition) for NATS Streaming

[![Go Report Card](https://goreportcard.com/badge/github.com/openfaas/nats-queue-worker)](https://goreportcard.com/badge/github.com/openfaas/nats-queue-worker)
[![Build Status](https://travis-ci.com/openfaas/nats-queue-worker.svg?branch=master)](https://travis-ci.com/openfaas/nats-queue-worker)

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/openfaas/nats-queue-worker?tab=overview)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)
[![Derek App](https://alexellis.o6s.io/badge?owner=openfaas&repo=nats-queue-worker)](https://github.com/alexellis/derek/)

The queue-worker (Community Edition) processes asynchronous function invocation requests, you can read more about this in the [async documentation](https://docs.openfaas.com/reference/async/)

## Usage

Screenshots from keynote / video - find out more over at https://www.openfaas.com/

<img width="1440" alt="screen shot 2017-10-26 at 15 55 25" src="https://user-images.githubusercontent.com/6358735/32060207-049d4afa-ba66-11e7-8fc2-f4a0a84cbdaf.png">

<img width="1440" alt="screen shot 2017-10-26 at 15 55 19" src="https://user-images.githubusercontent.com/6358735/32060206-047eb75c-ba66-11e7-94d3-1343ea1811db.png">

<img width="1440" alt="screen shot 2017-10-26 at 15 55 06" src="https://user-images.githubusercontent.com/6358735/32060205-04545692-ba66-11e7-9e6d-b800a07b9bf5.png">

### Configuration

| Parameter               | Description                           | Default                                                    |
| ----------------------- | ----------------------------------    | ---------------------------------------------------------- |
| `write_debug` | Print verbose logs | `false` |
| `faas_gateway_address` | Address of gateway DNS name | `gateway` |
| `faas_gateway_port` | Port of gateway service | `8080` |
| `faas_max_reconnect` | An integer of the amount of reconnection attempts when the NATS connection is lost | `120` |
| `faas_nats_address` | The host at which NATS Streaming can be reached | `nats` |
| `faas_nats_port` | The port at which NATS Streaming can be reached | `4222` |
| `faas_nats_cluster_name` | The name of the target NATS Streaming cluster | `faas-cluster` |
| `faas_reconnect_delay` | Delay between retrying to connect to NATS | `2s` |
| `faas_print_body` | Print the body of the function invocation | `false` |
