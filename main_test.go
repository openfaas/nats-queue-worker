package main

import (
	"testing"

	ftypes "github.com/openfaas/faas-provider/types"
)

func Test_makeFunctionURL_DefaultPathQS_GatewayInvoke_IncludesGWAddress(t *testing.T) {
	config := QueueWorkerConfig{
		FunctionSuffix: "",
		GatewayInvoke:  true,
		GatewayAddress: "gateway",
		GatewayPort:    8080,
	}
	req := ftypes.QueueRequest{
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
	req := ftypes.QueueRequest{
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
	req := ftypes.QueueRequest{
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
	req := ftypes.QueueRequest{
		Function: "function1",
		Path:     "/",
	}

	fnURL := makeFunctionURL(&req, &config, req.Path, req.QueryString)

	wantURL := "http://function1.openfaas-fn:8080/"
	if fnURL != wantURL {
		t.Errorf("want %s, got %s", wantURL, fnURL)
	}
}
