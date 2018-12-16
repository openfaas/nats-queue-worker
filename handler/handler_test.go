package handler

import (
	"github.com/openfaas/nats-queue-worker/nats"
	"os"
	"strings"
	"testing"
)

func Test_GetClientID_ContainsHostname(t *testing.T) {
	c := DefaultNatsConfig{}

	val := c.GetClientID()

	hostname, _ := os.Hostname()
	encodedHostname := nats.GetClientID(hostname)
	if !strings.HasSuffix(val, encodedHostname) {
		t.Errorf("GetClientID should contain hostname as suffix, got: %s", val)
		t.Fail()
	}
}

func TestCreategetClientID(t *testing.T) {
	clientID := getClientID("computer-a")
	want := "faas-publisher-computer-a"
	if clientID != want {
		t.Logf("Want clientID: `%s`, but got: `%s`\n", want, clientID)
		t.Fail()
	}
}

func TestCreategetClientIDWhenHostHasUnsupportedCharacters(t *testing.T) {
	clientID := getClientID("computer-a.acme.com")
	want := "faas-publisher-computer-a_acme_com"
	if clientID != want {
		t.Logf("Want clientID: `%s`, but got: `%s`\n", want, clientID)
		t.Fail()
	}
}
