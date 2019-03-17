package http

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"github.com/go-fed/httpsig"
	"net/http"
	"strconv"
	"testing"
)

var(
	testRSAPrivateKey *rsa.PrivateKey
	testRSAPublicKey  *rsa.PublicKey
)

func init()  {
	var err error
	testRSAPrivateKey, err = loadPrivateKey([]byte(testPrivateKey))
	if err != nil {
		panic(err)
	}

	testRSAPublicKey , err = loadPublicKey([]byte(testPublicKey))
	if err != nil {
		panic(err)
	}
}

func TestSignMessage(t *testing.T) {
	type args struct {
		request *http.Request
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		eval    func(r *http.Request) error
	}{
		{
			name: "All headers test", // https://tools.ietf.org/html/draft-cavage-http-signatures-10#appendix-C.1
			args: struct{ request *http.Request
			}{
				request: createRequestWithDateAndContentType("POST", "http://example.com/foo?param=value&pet=dog", `{"hello": "world"}`, "Sun, 05 Jan 2014 21:31:40 GMT", "application/json"),
			},
			wantErr: false,
			eval: func(r *http.Request) error {
				signature := r.Header["Authorization"][0]
				expectedSignature := `Signature keyId="callback",algorithm="rsa-sha256",headers="(request-target) host date content-type digest content-length",signature="vSdrb+dS3EceC9bcwHSo4MlyKS59iFIrhgYkz8+oVLEEzmYZZvRs8rgOp+63LEM3v+MFHB32NfpB2bEKBIvB1q52LaEUHFv120V01IL+TAD48XaERZFukWgHoBTLMhYS2Gb51gWxpeIq8knRmPnYePbF5MOkR0Zkly4zKH7s1dE="`
				if signature != expectedSignature {
					return fmt.Errorf("\nwant: %s\ngot:  %s", expectedSignature, signature)
				}

				return nil
			},
		},
		{
			name: "request is missing date and digest header, should be added automatically",
			args: struct{ request *http.Request
			}{
				request: createRequest("POST", "http://callback.com", `{ "name": "foo"}`),
			},
			wantErr: false,
			eval: func(r *http.Request) error {
				signature := r.Header["Authorization"]
				digest := r.Header["Digest"]
				if len (signature) == 0 {
					return fmt.Errorf("signature not present")
				}

				if len (digest) == 0 {
					return fmt.Errorf("digest not present")
				}

				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := signMessageWithKey(testRSAPrivateKey, tt.args.request, true); (err != nil) != tt.wantErr {
				t.Fatalf("signMessageWithKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err:= tt.eval(tt.args.request); err != nil {
				t.Errorf("error validating signature. %v", err)
			}
		})
	}
}

func TestVerifyMessage(t *testing.T) {
	r := createRequest("POST", "http://callback.com", `{ "name": "foo"}`)
	if err := signMessageWithKey(testRSAPrivateKey, r, true); err != nil {
		t.Errorf("signMessageWithKey() error = %v", err)
	}

	v, err := httpsig.NewVerifier(r)
	if err != nil {
		t.Errorf("httpsig.NewVerifier(r) error = %v", err)
	}

	if err := v.Verify(testRSAPublicKey, httpsig.RSA_SHA256); err != nil {
		t.Errorf("error verifying request error = %v", err)
	}
}

func TestLoadPrivateKey(t *testing.T) {
	key, err := loadPrivateKey([]byte(testPrivateKey))
	if err != nil {
		t.Errorf("error loading private key from PEM. %v", err)
	}

	publicKey := key.Public()
	if publicKey == nil {
		t.Errorf("error creating public key from private")
	}
}

func TestCreateBodyDigest(t *testing.T) {
	r, err := http.NewRequest("POST", "/create", bytes.NewBuffer([]byte(testBody)))
	if err != nil {
		t.Error(err)
	}

	setDigest(r)

	if r.Header["Digest"][0] != testBodySHA254 {
		t.Errorf("want %s got %s", testBodySHA254, r.Header["Digest"][0])
	}
}

func createRequest(method string, url string, body string) *http.Request {
	r, _ := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	r.Header["Host"] = []string{r.URL.Host}
	r.Header["Content-Length"] = []string{strconv.Itoa(len(body))}
	return r
}

func createRequestWithDateAndContentType(method string, url string, body string, time string, contentType string) *http.Request {
	r := createRequest(method, url, body)
	r.Header["Date"] = []string{time}
	r.Header["Content-Type"] = []string{contentType}
	return r
}

const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDCFENGw33yGihy92pDjZQhl0C36rPJj+CvfSC8+q28hxA161QF
NUd13wuCTUcq0Qd2qsBe/2hFyc2DCJJg0h1L78+6Z4UMR7EOcpfdUE9Hf3m/hs+F
UR45uBJeDK1HSFHD8bHKD6kv8FPGfJTotc+2xjJwoYi+1hqp1fIekaxsyQIDAQAB
AoGBAJR8ZkCUvx5kzv+utdl7T5MnordT1TvoXXJGXK7ZZ+UuvMNUCdN2QPc4sBiA
QWvLw1cSKt5DsKZ8UETpYPy8pPYnnDEz2dDYiaew9+xEpubyeW2oH4Zx71wqBtOK
kqwrXa/pzdpiucRRjk6vE6YY7EBBs/g7uanVpGibOVAEsqH1AkEA7DkjVH28WDUg
f1nqvfn2Kj6CT7nIcE3jGJsZZ7zlZmBmHFDONMLUrXR/Zm3pR5m0tCmBqa5RK95u
412jt1dPIwJBANJT3v8pnkth48bQo/fKel6uEYyboRtA5/uHuHkZ6FQF7OUkGogc
mSJluOdc5t6hI1VsLn0QZEjQZMEOWr+wKSMCQQCC4kXJEsHAve77oP6HtG/IiEn7
kpyUXRNvFsDE0czpJJBvL/aRFUJxuRK91jhjC68sA7NsKMGg5OXb5I5Jj36xAkEA
gIT7aFOYBFwGgQAQkWNKLvySgKbAZRTeLBacpHMuQdl1DfdntvAyqpAZ0lY0RKmW
G6aFKaqQfOXKCyWoUiVknQJAXrlgySFci/2ueKlIE1QqIiLSZ8V8OlpFLRnb1pzI
7U1yQXnTAEFYM560yJlzUpOb1V4cScGd365tiSMvxLOvTA==
-----END RSA PRIVATE KEY-----`

const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDCFENGw33yGihy92pDjZQhl0C3
6rPJj+CvfSC8+q28hxA161QFNUd13wuCTUcq0Qd2qsBe/2hFyc2DCJJg0h1L78+6
Z4UMR7EOcpfdUE9Hf3m/hs+FUR45uBJeDK1HSFHD8bHKD6kv8FPGfJTotc+2xjJw
oYi+1hqp1fIekaxsyQIDAQAB
-----END PUBLIC KEY-----`

const testBody = `{"hello": "world"}`
const testBodySHA254 = `SHA-256=X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=`