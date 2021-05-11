package main

import (
	"strings"
	"testing"

	stan "github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"github.com/openfaas/faas/gateway/queue"
)

func Test_makeFunctionURL_DefaultPathQS_GatewayInvoke_IncludesGWAddress(t *testing.T) {
	config := QueueWorkerConfig{
		FunctionSuffix: "",
		GatewayInvoke:  true,
		GatewayAddress: "gateway",
		GatewayPort:    8080,
	}
	req := queue.Request{
		Function: "function1",
		Path:     "/",
	}

	fnURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)
	wantURL := "http://gateway:8080/function/function1/"
	if fnURL != wantURL {
		t.Errorf("want %s, got %s", wantURL, fnURL)
	}
}

func Test_makeFunctionURL_DefaultPathQS_GatewayInvoke_WithQS(t *testing.T) {
	config := QueueWorkerConfig{
		FunctionSuffix: "",
		GatewayInvoke:  true,
		GatewayAddress: "gateway",
		GatewayPort:    8080,
	}
	req := queue.Request{
		Function:    "function1",
		QueryString: "user=1",
	}

	fnURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)
	wantURL := "http://gateway:8080/function/function1/?user=1"
	if fnURL != wantURL {
		t.Errorf("want %s, got %s", wantURL, fnURL)
	}
}

func Test_makeFunctionURL_DefaultPathQS_GatewayInvoke_WithPath(t *testing.T) {
	config := QueueWorkerConfig{
		FunctionSuffix: "",
		GatewayInvoke:  true,
		GatewayAddress: "gateway",
		GatewayPort:    8080,
	}
	req := queue.Request{
		Function: "function1",
		Path:     "/resources/main.css",
	}

	fnURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)
	wantURL := "http://gateway:8080/function/function1/resources/main.css"
	if fnURL != wantURL {
		t.Errorf("want %s, got %s", wantURL, fnURL)
	}
}

func Test_makeFunctionURL_DefaultPathQS_GatewayInvokeOff_UsesDirectInvocation(t *testing.T) {
	config := QueueWorkerConfig{
		FunctionSuffix: ".openfaas-fn",
		GatewayInvoke:  false,
		GatewayAddress: "gateway",
		GatewayPort:    8080,
	}
	req := queue.Request{
		Function: "function1",
		Path:     "/",
	}

	fnURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)

	wantURL := "http://function1.openfaas-fn:8080/"
	if fnURL != wantURL {
		t.Errorf("want %s, got %s", wantURL, fnURL)
	}
}

func Test_redact(t *testing.T) {

	m := &stan.Msg{
		MsgProto: pb.MsgProto{
			Data: []byte(`to-be-redacted-in-logs`),
		},
	}
	want := &stan.Msg{
		MsgProto: pb.MsgProto{
			Data: []byte(`xxxxxx`),
		},
	}
	got := redact(m)

	if strings.Compare(got, want.String()) != 0 {
		t.Errorf("want %s, got %s", want, got)
	}
}
