package osv

import (
	"crypto/ed25519"
	"encoding/json"
)

// BuildSignedBundle serializes advisories into the canonical maldb.json format
// and signs it with privKey. Used by offline release scripts and tests.
func BuildSignedBundle(version string, advisories []Advisory, privKey ed25519.PrivateKey) (payload, sig []byte, err error) {
	doc := dbDoc{
		Version:    version,
		Advisories: advisories,
	}
	payload, err = json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	sig, err = SignBundle(payload, privKey)
	if err != nil {
		return nil, nil, err
	}
	return payload, sig, nil
}
