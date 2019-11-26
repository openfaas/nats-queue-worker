package main

import (
	"os"
	"testing"
	"time"
)

func Test_ReadConfig_GatewayInvokeDefault(t *testing.T) {

	readConfig := ReadConfig{}

	os.Setenv("gateway_invoke", "")
	cfg, _ := readConfig.Read()

	gatewayInvokeWant := false
	if cfg.GatewayInvoke != gatewayInvokeWant {
		t.Errorf("gatewayInvokeWant want %v, got %v", gatewayInvokeWant, cfg.GatewayInvoke)
	}
}

func Test_ReadConfig_GatewayInvokeSetToTrue(t *testing.T) {

	readConfig := ReadConfig{}

	os.Setenv("gateway_invoke", "true")
	cfg, _ := readConfig.Read()

	gatewayInvokeWant := true
	if cfg.GatewayInvoke != gatewayInvokeWant {
		t.Errorf("gatewayInvokeWant want %v, got %v", gatewayInvokeWant, cfg.GatewayInvoke)
	}
}

func Test_ReadConfig_BasicAuthDefaultIsFalse(t *testing.T) {
	readConfig := ReadConfig{}

	os.Setenv("basic_auth", "")
	cfg, _ := readConfig.Read()

	want := false
	if cfg.BasicAuth != want {
		t.Errorf("basicAuth want %v, got %v", want, cfg.BasicAuth)
	}
}

func Test_ReadConfig_BasicAuthSetToTrue(t *testing.T) {
	readConfig := ReadConfig{}

	os.Setenv("basic_auth", "true")
	cfg, _ := readConfig.Read()

	want := true
	if cfg.BasicAuth != want {
		t.Errorf("basicAuth want %v, got %v", want, cfg.BasicAuth)
	}
}

func Test_ReadConfig_IncorrectPortValue(t *testing.T) {

	readConfig := ReadConfig{}

	os.Setenv("faas_gateway_port", "8080a")
	_, err := readConfig.Read()
	want := "converting faas_gateway_port 8080a to int error: strconv.Atoi: parsing \"8080a\": invalid syntax"

	if err.Error() != want {
		t.Errorf("basicfaas_gateway_port want %q, got %q", want, err.Error())

	}
}

func Test_ReadConfig(t *testing.T) {

	readConfig := ReadConfig{}

	os.Setenv("faas_nats_address", "test_nats")
	os.Setenv("faas_nats_port", "1234")
	os.Setenv("faas_nats_cluster_name", "example-nats-cluster")
	os.Setenv("faas_nats_durable_queue_subscription", "true")
	os.Setenv("faas_gateway_address", "test_gatewayaddr")
	os.Setenv("faas_gateway_port", "8080")
	os.Setenv("faas_function_suffix", "test_suffix")
	os.Setenv("faas_print_body", "true")
	os.Setenv("write_debug", "true")
	os.Setenv("max_inflight", "10")
	os.Setenv("ack_wait", "10ms")

	config, _ := readConfig.Read()

	want := "test_nats"
	if config.NatsAddress != want {
		t.Logf("NatsAddress want `%s`, got `%s`\n", want, config.NatsAddress)
		t.Fail()
	}

	wantNatsPort := 1234
	if config.NatsPort != wantNatsPort {
		t.Logf("NatsPort want `%d`, got `%d`\n", wantNatsPort, config.NatsPort)
		t.Fail()
	}

	wantNatsClusterName := "example-nats-cluster"
	if config.NatsClusterName != wantNatsClusterName {
		t.Logf("NatsClusterName want `%s`, got `%s`\n", wantNatsClusterName, config.NatsClusterName)
		t.Fail()
	}

	wantNatsDurableQueueSubscription := true
	if config.NatsDurableQueueSubscription != wantNatsDurableQueueSubscription {
		t.Logf("NatsDurableQueueSubscription want `%t`, got `%t`\n", wantNatsDurableQueueSubscription, config.NatsDurableQueueSubscription)
		t.Fail()
	}

	want = "test_gatewayaddr"
	if config.GatewayAddress != want {
		t.Logf("GatewayAddress want `%s`, got `%s`\n", want, config.GatewayAddress)
		t.Fail()
	}

	wantGatewayPort := 8080
	if config.GatewayPort != wantGatewayPort {
		t.Logf("GatewayPort want `%d`, got `%d`\n", wantGatewayPort, config.GatewayPort)
		t.Fail()
	}

	want = "test_gatewayaddr:8080"
	if config.GatewayAddressURL() != want {
		t.Logf("GatewayAddressURL want `%s`, got `%s`\n", want, config.GatewayAddressURL())
		t.Fail()
	}

	want = "test_suffix"
	if config.FunctionSuffix != want {
		t.Logf("FunctionSuffix want `%s`, got `%s`\n", want, config.FunctionSuffix)
		t.Fail()
	}

	if config.DebugPrintBody != true {
		t.Logf("DebugPrintBody want `%v`, got `%v`\n", true, config.DebugPrintBody)
		t.Fail()
	}

	if config.WriteDebug != true {
		t.Logf("WriteDebug want `%v`, got `%v`\n", true, config.WriteDebug)
		t.Fail()
	}

	wantMaxInflight := 10
	if config.MaxInflight != wantMaxInflight {
		t.Logf("maxInflight want `%v`, got `%v`\n", wantMaxInflight, config.MaxInflight)
		t.Fail()
	}

	wantAckWait := time.Millisecond * 10
	if config.AckWait != wantAckWait {
		t.Logf("maxInflight want `%v`, got `%v`\n", wantAckWait, config.AckWait)
		t.Fail()
	}

	os.Unsetenv("max_inflight")
	os.Unsetenv("ack_wait")

	config, _ = readConfig.Read()

	wantMaxInflight = 1
	if config.MaxInflight != wantMaxInflight {
		t.Logf("maxInflight want `%v`, got `%v`\n", wantMaxInflight, config.MaxInflight)
		t.Fail()
	}

	wantAckWait = time.Second * 30
	if config.AckWait != wantAckWait {
		t.Logf("maxInflight want `%v`, got `%v`\n", wantAckWait, config.AckWait)
		t.Fail()
	}

	os.Setenv("max_inflight", "10.00")
	os.Setenv("ack_wait", "10")

	config, _ = readConfig.Read()

	wantMaxInflight = 1
	if config.MaxInflight != wantMaxInflight {
		t.Logf("maxInflight want `%v`, got `%v`\n", wantMaxInflight, config.MaxInflight)
		t.Fail()
	}

	wantAckWait = time.Second * 30
	if config.AckWait != wantAckWait {
		t.Logf("ackWait want `%v`, got `%v`\n", wantAckWait, config.AckWait)
		t.Fail()
	}
}
