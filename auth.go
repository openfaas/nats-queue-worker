package main

import (
	"fmt"
	"github.com/openfaas/faas-provider/auth"
)

//LoadCredentials load credentials from dis
func LoadCredentials(secretMountPath string) (*auth.BasicAuthCredentials, error) {
	reader := auth.ReadBasicAuthFromDisk{
		SecretMountPath: secretMountPath,
	}

	credentials, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("unable to read basic auth: %s", err.Error())
	}
	return credentials, nil
}
