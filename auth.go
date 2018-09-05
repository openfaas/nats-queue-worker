package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/openfaas/faas/gateway/types"
)

//AddBasicAuth to a request by reading secrets
func AddBasicAuth(req *http.Request) error {
	reader := types.ReadBasicAuthFromDisk{}

	if len(os.Getenv("secret_mount_path")) > 0 {
		reader.SecretMountPath = os.Getenv("secret_mount_path")
	}

	credentials, err := reader.Read()
	if err != nil {
		return fmt.Errorf("Unable to read basic auth: %s", err.Error())
	}

	req.SetBasicAuth(credentials.User, credentials.Password)
	return nil
}
