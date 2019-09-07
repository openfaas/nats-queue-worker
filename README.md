## Queue worker for OpenFaaS - NATS Streaming

[![Build Status](https://travis-ci.org/openfaas/nats-queue-worker.svg?branch=master)](https://travis-ci.org/openfaas/nats-queue-worker)

This is a queue-worker to enable asynchronous processing of function requests. 

> Note: A Kafka queue-worker is under-way through a PR on the main OpenFaaS repository.

* [Read more in the async guide](https://docs.openfaas.com/reference/async/)

Hub image: [openfaas/queue-worker](https://hub.docker.com/r/openfaas/queue-worker/)

License: MIT

Screenshots from keynote / video - find out more over at https://www.openfaas.com/

<img width="1440" alt="screen shot 2017-10-26 at 15 55 25" src="https://user-images.githubusercontent.com/6358735/32060207-049d4afa-ba66-11e7-8fc2-f4a0a84cbdaf.png">

<img width="1440" alt="screen shot 2017-10-26 at 15 55 19" src="https://user-images.githubusercontent.com/6358735/32060206-047eb75c-ba66-11e7-94d3-1343ea1811db.png">

<img width="1440" alt="screen shot 2017-10-26 at 15 55 06" src="https://user-images.githubusercontent.com/6358735/32060205-04545692-ba66-11e7-9e6d-b800a07b9bf5.png">

### Configuration

| Parameter               | Description                           | Default                                                    |
| ----------------------- | ----------------------------------    | ---------------------------------------------------------- |
| `gateway_invoke` | When `true` functions are invoked via the gateway, when `false` they are invoked directly | `false` |
| `basic_auth` | When `true` basic auth is used to post any function statistics back to the gateway | `false` |
| `write_debug` | Print verbose logs | `false` |
| `faas_gateway_address` | Address of gateway DNS name | `gateway` |
| `faas_gateway_port` | Port of gateway service | `8080` |
| `faas_function_suffix` | When `gateway_invoke` is `false`, this suffix is used to contact a function, it may correspond to a Kubernetes namespace  | `` |
| `faas_max_reconnect` | An integer of the amount of reconnection attempts when the NATS connection is lost | `120` |
| `faas_nats_address` | The DNS entry for NATS | `nats` |
| `faas_reconnect_delay` | Delay between retrying to connect to NATS | `2s` |
| `faas_print_body` | Print the body of the function invocation | `false` |
