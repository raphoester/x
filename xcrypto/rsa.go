package xcrypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
)

func NewPublicRSAFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("key not found in PEM block")
	}

	var key *rsa.PublicKey
	switch block.Type {
	case "RSA PUBLIC KEY", "PUBLIC KEY":
		rsaKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse RSA public key")
		}

		cast, ok := rsaKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.Errorf("expected *rsa.PublicKey, got %T", rsaKey)
		}

		key = cast
	default:
		return nil, fmt.Errorf("unsupported public key type '%s'", block.Type)
	}

	return key, nil
}
