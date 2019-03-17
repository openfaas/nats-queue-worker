// Implements HTTP request and response signing and verification. Supports the
// major MAC and asymmetric key signature algorithms. It has several safety
// restrictions: One, none of the widely known non-cryptographically safe
// algorithms are permitted; Two, the RSA SHA256 algorithms must be available in
// the binary (and it should, barring export restrictions); Finally, the library
// assumes either the 'Authorizationn' or 'Signature' headers are to be set (but
// not both).
package httpsig

import (
	"crypto"
	"fmt"
	"net/http"
)

// Algorithm specifies a cryptography secure algorithm for signing HTTP requests
// and responses.
type Algorithm string

const (
	// MAC-based algoirthms.
	HMAC_SHA224      Algorithm = hmacPrefix + "-" + sha224String
	HMAC_SHA256      Algorithm = hmacPrefix + "-" + sha256String
	HMAC_SHA384      Algorithm = hmacPrefix + "-" + sha384String
	HMAC_SHA512      Algorithm = hmacPrefix + "-" + sha512String
	HMAC_RIPEMD160   Algorithm = hmacPrefix + "-" + ripemd160String
	HMAC_SHA3_224    Algorithm = hmacPrefix + "-" + sha3_224String
	HMAC_SHA3_256    Algorithm = hmacPrefix + "-" + sha3_256String
	HMAC_SHA3_384    Algorithm = hmacPrefix + "-" + sha3_384String
	HMAC_SHA3_512    Algorithm = hmacPrefix + "-" + sha3_512String
	HMAC_SHA512_224  Algorithm = hmacPrefix + "-" + sha512_224String
	HMAC_SHA512_256  Algorithm = hmacPrefix + "-" + sha512_256String
	HMAC_BLAKE2S_256 Algorithm = hmacPrefix + "-" + blake2s_256String
	HMAC_BLAKE2B_256 Algorithm = hmacPrefix + "-" + blake2b_256String
	HMAC_BLAKE2B_384 Algorithm = hmacPrefix + "-" + blake2b_384String
	HMAC_BLAKE2B_512 Algorithm = hmacPrefix + "-" + blake2b_512String
	BLAKE2S_256      Algorithm = blake2s_256String
	BLAKE2B_256      Algorithm = blake2b_256String
	BLAKE2B_384      Algorithm = blake2b_384String
	BLAKE2B_512      Algorithm = blake2b_512String
	// RSA-based algorithms.
	RSA_SHA224 Algorithm = rsaPrefix + "-" + sha224String
	// RSA_SHA256 is the default algorithm.
	RSA_SHA256    Algorithm = rsaPrefix + "-" + sha256String
	RSA_SHA384    Algorithm = rsaPrefix + "-" + sha384String
	RSA_SHA512    Algorithm = rsaPrefix + "-" + sha512String
	RSA_RIPEMD160 Algorithm = rsaPrefix + "-" + ripemd160String
	// Just because you can glue things together, doesn't mean they will
	// work. The following options are not supported.
	rsa_SHA3_224    Algorithm = rsaPrefix + "-" + sha3_224String
	rsa_SHA3_256    Algorithm = rsaPrefix + "-" + sha3_256String
	rsa_SHA3_384    Algorithm = rsaPrefix + "-" + sha3_384String
	rsa_SHA3_512    Algorithm = rsaPrefix + "-" + sha3_512String
	rsa_SHA512_224  Algorithm = rsaPrefix + "-" + sha512_224String
	rsa_SHA512_256  Algorithm = rsaPrefix + "-" + sha512_256String
	rsa_BLAKE2S_256 Algorithm = rsaPrefix + "-" + blake2s_256String
	rsa_BLAKE2B_256 Algorithm = rsaPrefix + "-" + blake2b_256String
	rsa_BLAKE2B_384 Algorithm = rsaPrefix + "-" + blake2b_384String
	rsa_BLAKE2B_512 Algorithm = rsaPrefix + "-" + blake2b_512String
)

// HTTP Signatures can be applied to different HTTP headers, depending on the
// expected application behavior.
type SignatureScheme string

const (
	// Signature will place the HTTP Signature into the 'Signature' HTTP
	// header.
	Signature SignatureScheme = "Signature"
	// Authorization will place the HTTP Signature into the 'Authorization'
	// HTTP header.
	Authorization SignatureScheme = "Authorization"
)

const (
	// The HTTP Signatures specification uses the "Signature" auth-scheme
	// for the Authorization header. This is coincidentally named, but not
	// semantically the same, as the "Signature" HTTP header value.
	signatureAuthScheme = "Signature"
)

// There are subtle differences to the values in the header. The Authorization
// header has an 'auth-scheme' value that must be prefixed to the rest of the
// key and values.
func (s SignatureScheme) authScheme() string {
	switch s {
	case Authorization:
		return signatureAuthScheme
	default:
		return ""
	}
}

// Signers will sign HTTP requests or responses based on the algorithms and
// headers selected at creation time.
//
// Signers are not safe to use between multiple goroutines.
//
// Note that signatures do set the deprecated 'algorithm' parameter for
// backwards compatibility.
type Signer interface {
	// SignRequest signs the request using a private key. The public key id
	// is used by the HTTP server to identify which key to use to verify the
	// signature.
	//
	// If the Signer was created using a MAC based algorithm, then the key
	// is expected to be of type []byte. If the Signer was created using an
	// RSA based algorithm, then the private key is expected to be of type
	// *rsa.PrivateKey.
	SignRequest(pKey crypto.PrivateKey, pubKeyId string, r *http.Request) error
	// SignResponse signs the response using a private key. The public key
	// id is used by the HTTP client to identify which key to use to verify
	// the signature.
	//
	// If the Signer was created using a MAC based algorithm, then the key
	// is expected to be of type []byte. If the Signer was created using an
	// RSA based algorithm, then the private key is expected to be of type
	// *rsa.PrivateKey.
	SignResponse(pKey crypto.PrivateKey, pubKeyId string, r http.ResponseWriter) error
}

// NewSigner creates a new Signer with the provided algorithm preferences to
// make HTTP signatures. Only the first available algorithm will be used, which
// is returned by this function along with the Signer. If none of the preferred
// algorithms were available, then the default algorithm is used. The headers
// specified will be included into the HTTP signatures.
//
// The provided scheme determines which header is populated with the HTTP
// Signature.
//
// An error is returned if an unknown or a known cryptographically insecure
// Algorithm is provided.
func NewSigner(prefs []Algorithm, headers []string, scheme SignatureScheme) (Signer, Algorithm, error) {
	for _, pref := range prefs {
		s, err := newSigner(pref, headers, scheme)
		if err != nil {
			continue
		}
		return s, pref, err
	}
	s, err := newSigner(defaultAlgorithm, headers, scheme)
	return s, defaultAlgorithm, err
}

// Verifier verifies HTTP Signatures.
//
// It will determine which of the supported headers has the parameters
// that define the signature.
//
// Verifiers are not safe to use between multiple goroutines.
//
// Note that verification ignores the deprecated 'algorithm' parameter.
type Verifier interface {
	// KeyId gets the public key id that the signature is signed with.
	//
	// Note that the application is expected to determine the algorithm
	// used based on metadata or out-of-band information for this key id.
	KeyId() string
	// Verify accepts the public key specified by KeyId and returns an
	// error if verification fails or if the signature is malformed. The
	// algorithm must be the one used to create the signature in order to
	// pass verification. The algorithm is determined based on metadata or
	// out-of-band information for the key id.
	//
	// If the signature was created using a MAC based algorithm, then the
	// key is expected to be of type []byte. If the signature was created
	// using an RSA based algorithm, then the public key is expected to be
	// of type *rsa.PublicKey.
	Verify(pKey crypto.PublicKey, algo Algorithm) error
}

// NewVerifier verifies the given request. It returns an error if the HTTP
// Signature parameters are not present in any headers, are present in more than
// one header, are malformed, or are missing required parameters. It ignores
// unknown HTTP Signature parameters.
func NewVerifier(r *http.Request) (Verifier, error) {
	return newVerifier(r.Header, func(h http.Header, toInclude []string) (string, error) {
		return signatureString(h, toInclude, addRequestTarget(r))
	})
}

// NewResponseVerifier verifies the given response. It returns errors under the
// same conditions as NewVerifier.
func NewResponseVerifier(r *http.Response) (Verifier, error) {
	return newVerifier(r.Header, func(h http.Header, toInclude []string) (string, error) {
		return signatureString(h, toInclude, requestTargetNotPermitted)
	})
}

func newSigner(algo Algorithm, headers []string, scheme SignatureScheme) (Signer, error) {
	s, err := signerFromString(string(algo))
	if err == nil {
		a := &asymmSigner{
			s:            s,
			headers:      headers,
			targetHeader: scheme,
			prefix:       scheme.authScheme(),
		}
		return a, nil
	}
	m, err := macerFromString(string(algo))
	if err != nil {
		return nil, fmt.Errorf("no crypto implementation available for %q", algo)
	}
	c := &macSigner{
		m:            m,
		headers:      headers,
		targetHeader: scheme,
		prefix:       scheme.authScheme(),
	}
	return c, nil
}
