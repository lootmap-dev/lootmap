package osv_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/osv"
)

func TestLockfileParsersCatchKnownMaliciousVersions(t *testing.T) {
	db, err := osv.Default(nil)
	if err != nil {
		t.Fatal(err)
	}

	npmLock := []byte(`{
  "name": "my-app",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "my-app", "version": "1.0.0"},
    "node_modules/@ctrl/tinycolor": {"version": "4.1.1"},
    "node_modules/lodash": {"version": "4.17.21"}
  }
}`)
	npmRefs := osv.ParsePackageLock(npmLock)
	foundNPM := false
	for _, r := range npmRefs {
		if adv, ok := db.Lookup(r.Name, r.Version); ok && adv.ID == "MAL-2025-4912" {
			foundNPM = true
		}
	}
	if !foundNPM {
		t.Fatalf("expected @ctrl/tinycolor@4.1.1 to match MAL-2025-4912 in package-lock.json, refs=%+v", npmRefs)
	}

	pnpmLock := []byte(`lockfileVersion: '9.0'
packages:
  '@ctrl/tinycolor@4.1.2':
    resolution: {integrity: sha512-xxx}
  'react@18.3.1':
    resolution: {integrity: sha512-yyy}
`)
	pnpmRefs := osv.ParsePNPMLock(pnpmLock)
	foundPNPM := false
	for _, r := range pnpmRefs {
		if adv, ok := db.Lookup(r.Name, r.Version); ok && adv.ID == "MAL-2025-4912" {
			foundPNPM = true
		}
	}
	if !foundPNPM {
		t.Fatalf("expected @ctrl/tinycolor@4.1.2 to match MAL-2025-4912 in pnpm-lock.yaml, refs=%+v", pnpmRefs)
	}

	yarnLock := []byte(`# yarn lockfile v1

"event-stream-malicious-test@^3.3.0":
  version "3.3.6"
  resolved "https://registry.yarnpkg.com/event-stream-malicious-test/-/event-stream-malicious-test-3.3.6.tgz"
`)
	yarnRefs := osv.ParseYarnLock(yarnLock)
	foundYarn := false
	for _, r := range yarnRefs {
		if adv, ok := db.Lookup(r.Name, r.Version); ok && adv.ID == "MAL-2026-1104" {
			foundYarn = true
		}
	}
	if !foundYarn {
		t.Fatalf("expected event-stream-malicious-test@3.3.6 to match MAL-2026-1104 in yarn.lock, refs=%+v", yarnRefs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBuildVerifyAndDownloadSignedBundle(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	advisories := []osv.Advisory{
		{
			ID:        "MAL-2026-9999",
			Ecosystem: "npm",
			Name:      "bad-pkg",
			Versions:  []string{"1.0.0"},
			Summary:   "Test malicious package",
		},
	}
	payload, sig, err := osv.BuildSignedBundle("2026-10-31", advisories, priv)
	if err != nil {
		t.Fatal(err)
	}

	// Tampered payload must fail signature verification.
	if err := osv.VerifyBundle(append([]byte(nil), payload...), []byte("bad-sig"), pub); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
	tampered := bytes.ReplaceAll(payload, []byte("bad-pkg"), []byte("ok--pkg"))
	if err := osv.VerifyBundle(tampered, sig, pub); err == nil {
		t.Fatal("expected tampered payload to fail Ed25519 verification")
	}

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body := payload
			if req.URL.Path == "/osv/maldb.json.sig" {
				body = sig
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	home := t.TempDir()
	db, err := osv.DownloadAndUpdate(context.Background(), osv.UpdateOptions{
		Home:      home,
		URL:       "https://releases.lootmap.dev/osv/maldb.json",
		PublicKey: pub,
		Client:    client,
	})
	if err != nil {
		t.Fatalf("DownloadAndUpdate: %v", err)
	}
	if _, ok := db.Lookup("bad-pkg", "1.0.0"); !ok {
		t.Fatal("expected updated DB to contain bad-pkg@1.0.0")
	}
}

func TestUpdateRefusesPublishedTestKey(t *testing.T) {
	if osv.DefaultPublicKeyHex != "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a" {
		t.Skip("a real release key is configured")
	}
	_, err := osv.DownloadAndUpdate(context.Background(), osv.UpdateOptions{Home: t.TempDir()})
	if !errors.Is(err, osv.ErrUpdateNotConfigured) {
		t.Fatalf("want ErrUpdateNotConfigured, got %v", err)
	}
}
