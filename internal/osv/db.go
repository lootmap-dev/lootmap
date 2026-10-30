// Package osv manages the embedded offline OSV MAL- advisory database,
// Ed25519 signature verification, lockfile parsers (npm, pnpm, yarn, npm cache),
// and the opt-in update-db downloader (the only code in lootmap allowed to
// import net/http).
package osv

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/lootmap-dev/lootmap/internal/env"
)

//go:embed maldb.json
var embeddedDBBytes []byte

// DefaultPublicKeyHex is the hex-encoded Ed25519 public key used to verify
// downloaded OSV MAL- database bundles.
//
// RELEASE BLOCKER: this is still the public RFC 8032 test-vector key, whose
// private half is published in the RFC. update-db refuses to use it (see
// ErrUpdateNotConfigured). Generate a real release key pair and replace this
// value before enabling update-db.
const DefaultPublicKeyHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"

// rfc8032TestKeyHex is the RFC 8032 section 7.1 TEST 1 public key. Anyone can
// sign with its private key, so it must never be trusted for real updates.
const rfc8032TestKeyHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"

// Advisory represents one OSV MAL- entry.
type Advisory struct {
	ID        string   `json:"id"`
	Ecosystem string   `json:"ecosystem"`
	Name      string   `json:"name"`
	Versions  []string `json:"versions"`
	Summary   string   `json:"summary"`
}

type dbDoc struct {
	Version    string     `json:"version"`
	Advisories []Advisory `json:"advisories"`
}

// DB indexes MAL- advisories by package name and version.
type DB struct {
	Version string
	byPkg   map[string]map[string]Advisory // name -> version -> Advisory
}

var (
	defaultOnce sync.Once
	defaultDB   *DB
)

// Default returns the embedded offline MAL- database, or a user-updated signed
// database from ~/.lootmap/maldb.json if present and valid.
func Default(e env.Env) (*DB, error) {
	if e != nil {
		if data, err := fs.ReadFile(e.FS(), ".lootmap/maldb.json"); err == nil {
			if db, err := ParseDB(data); err == nil {
				return db, nil
			}
		}
	}
	defaultOnce.Do(func() {
		defaultDB, _ = ParseDB(embeddedDBBytes)
	})
	if defaultDB == nil {
		return nil, errors.New("osv: failed to load embedded database")
	}
	return defaultDB, nil
}

// ParseDB parses JSON advisory database bytes into a DB.
func ParseDB(data []byte) (*DB, error) {
	var doc dbDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	byPkg := make(map[string]map[string]Advisory, len(doc.Advisories))
	for _, adv := range doc.Advisories {
		vers, ok := byPkg[adv.Name]
		if !ok {
			vers = make(map[string]Advisory, len(adv.Versions))
			byPkg[adv.Name] = vers
		}
		for _, v := range adv.Versions {
			vers[v] = adv
		}
	}
	return &DB{Version: doc.Version, byPkg: byPkg}, nil
}

// Lookup checks whether pkg@version is a known malicious release.
func (d *DB) Lookup(pkg, version string) (Advisory, bool) {
	if d == nil {
		return Advisory{}, false
	}
	if vers, ok := d.byPkg[pkg]; ok {
		if adv, ok := vers[version]; ok {
			return adv, true
		}
	}
	return Advisory{}, false
}

// VerifyBundle verifies an Ed25519 signature over payload using pubKey.
func VerifyBundle(payload, sig []byte, pubKey ed25519.PublicKey) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("osv: invalid public key length %d", len(pubKey))
	}
	if len(sig) != ed25519.SignatureSize || !ed25519.Verify(pubKey, payload, sig) {
		return errors.New("osv: signature verification failed")
	}
	return nil
}

// SignBundle signs payload with privKey (used by the offline DB generator and tests).
func SignBundle(payload []byte, privKey ed25519.PrivateKey) ([]byte, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return nil, errors.New("osv: invalid private key")
	}
	return ed25519.Sign(privKey, payload), nil
}

// DecodePublicKeyHex decodes a 64-char hex string into an ed25519.PublicKey.
func DecodePublicKeyHex(hexStr string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("osv: expected %d-byte public key, got %d", ed25519.PublicKeySize, len(b))
	}
	return ed25519.PublicKey(b), nil
}

// SaveVerifiedDB verifies sig against payload using pubKey, validates the JSON
// structure, and writes the database to ~/.lootmap/maldb.json.
func SaveVerifiedDB(home string, payload, sig []byte, pubKey ed25519.PublicKey) (*DB, error) {
	if err := VerifyBundle(payload, sig, pubKey); err != nil {
		return nil, err
	}
	db, err := ParseDB(payload)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".lootmap")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	target := filepath.Join(dir, "maldb.json")
	if err := os.WriteFile(target, payload, 0o600); err != nil {
		return nil, err
	}
	return db, nil
}
