package http

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-fed/httpsig"
)


type Signatures struct {
	debug bool
	secretsMountPath string
	privateKey *rsa.PrivateKey
}

func New(secretsMountPath string, debug bool) *Signatures {
	var privateKey *rsa.PrivateKey

	privateKeyPath := path.Join(secretsMountPath, "http-signing-private-key")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		log.Println("Http signatures disabled. Warning callback messages will not be signed missing private key: /run/secrets/http-signing-private-key")
	} else {

		privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
		if err != nil {
			panic(err)
		}

		privateKey, err = loadPrivateKey(privateKeyBytes)
		if err != nil {
			panic(err)
		}

		log.Printf("Http signatures enabled.")
	}

	return &Signatures{
		debug: debug,
		secretsMountPath: secretsMountPath,
		privateKey: privateKey,
	}
}

func (h *Signatures) SignMessage(request *http.Request) error {
	if h.privateKey == nil {
		return nil // backwards compatibility, if no signing key given we won't sign messages
	}

	return signMessageWithKey(h.privateKey, request, h.debug)
}

func signMessageWithKey(privateKey crypto.PrivateKey, r *http.Request, debug bool) error {
	if _, ok := r.Header["Host"]; !ok {
		r.Header["Host"] = []string{r.Host}
	}

	if _, ok := r.Header["Date"]; !ok {
		r.Header["Date"] = []string{time.Now().Format(http.TimeFormat)}
	}

	if _, ok := r.Header["Content-Type"]; !ok {
		r.Header["Content-Type"] = []string{"application/octet-stream"}
	}

	bodyBytes, err := setDigest(r)
	if err != nil {
		return err
	}

	headersToSign := []string{httpsig.RequestTarget, "host", "date", "content-type", "digest", "content-length"}
	preferences := []httpsig.Algorithm{httpsig.RSA_SHA256}

	signer, _, err := httpsig.NewSigner(preferences, headersToSign, httpsig.Authorization)
	if err != nil {
		return fmt.Errorf("error creating request signer. %v", err)
	}

	if err := signer.SignRequest(privateKey, "callback", r); err != nil {
		return fmt.Errorf("error siging request. %v", err)
	}

	if debug {
		debugPrintSignature(r, bodyBytes)
	}

	return nil
}

func setDigest(r *http.Request) ([]byte , error) {
	var bodyBytes []byte
	if _, ok := r.Header["Digest"]; !ok {
		body := ""
		if r.Body != nil {
			var err error
			bodyBytes, err = ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("error reading body. %v", err)
			}

			// And now set a new body, which will simulate the same data we read:
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			body = string(bodyBytes)
		}

		d := sha256.Sum256([]byte(body))
		r.Header["Digest"] = []string{fmt.Sprintf("SHA-256=%s", base64.StdEncoding.EncodeToString(d[:]))}

		if _, ok := r.Header["Content-Length"]; !ok {
			r.Header["Content-Length"] = []string{string(len(body))}
		}
	}

	return bodyBytes, nil
}

func debugPrintSignature(r *http.Request, body []byte) {
	log.Printf("--- queue worker signature debug ----\n")
	log.Printf("(request-target): %s %s\n", strings.ToLower(r.Method), requestPath(r))
	log.Printf("host: %s\n", strings.ToLower(r.Host))
	log.Printf("date: %s\n", headerOrDefault(r, "Date", ""))
	log.Printf("content-type: %s\n", headerOrDefault(r, "Content-Type", ""))
	log.Printf("digest: %s\n", headerOrDefault(r, "Digest", ""))
	log.Printf("content-length: %s\n", headerOrDefault(r, "Content-Length", ""))
	log.Printf("body:\n%s\n", string(body))
	log.Printf("authorization: %s\n", headerOrDefault(r , "Authorization", ""))
	log.Printf("--- queue worker signature debug ----\n")
}

func requestPath(r *http.Request) string {
	res := strings.Builder{}
	res.WriteString(r.URL.Path)

	if r.URL.RawQuery != "" {
		res.WriteString("?")
		res.WriteString(r.URL.RawQuery)
	}

	return res.String()
}

func headerOrDefault(r *http.Request, key string, defaultValue string) string {
	if v, ok := r.Header[key]; ok {
		return v[0]
	}

	return defaultValue
}

func loadPrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	pem, _ := pem.Decode(keyData)
	if pem.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("RSA private key is of the wrong type: %s", pem.Type)
	}

	return x509.ParsePKCS1PrivateKey(pem.Bytes)
}

func loadPublicKey(keyData []byte) (*rsa.PublicKey, error) {
	pem, _ := pem.Decode(keyData)
	if pem.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("public key is of the wrong type: %s", pem.Type)
	}

	key, err := x509.ParsePKIXPublicKey(pem.Bytes)
	if err != nil {
		return nil, err
	}

	return key.(*rsa.PublicKey), nil
}
