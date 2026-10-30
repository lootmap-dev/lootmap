package osv

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrUpdateNotConfigured is returned by DownloadAndUpdate when this build has no
// real release signing key yet, so a downloaded bundle could not be trusted.
var ErrUpdateNotConfigured = errors.New("update-db is not available in this build yet: no release signing key is configured; the built-in database is used")

// DefaultBundleURL is the canonical URL for the signed OSV MAL- database.
const DefaultBundleURL = "https://releases.lootmap.dev/osv/maldb.json"

// UpdateOptions configures the opt-in update-db command.
type UpdateOptions struct {
	Home      string
	URL       string
	PublicKey ed25519.PublicKey
	Client    *http.Client
}

// DownloadAndUpdate fetches the OSV database and its .sig signature, verifies
// the Ed25519 signature before touching disk, and saves ~/.lootmap/maldb.json.
func DownloadAndUpdate(ctx context.Context, opts UpdateOptions) (*DB, error) {
	url := opts.URL
	if url == "" {
		url = DefaultBundleURL
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	pub := opts.PublicKey
	if len(pub) == 0 {
		if DefaultPublicKeyHex == rfc8032TestKeyHex {
			return nil, ErrUpdateNotConfigured
		}
		var err error
		pub, err = DecodePublicKeyHex(DefaultPublicKeyHex)
		if err != nil {
			return nil, err
		}
	}

	payload, err := fetchBytes(ctx, client, url)
	if err != nil {
		return nil, fmt.Errorf("download db: %w", err)
	}
	sig, err := fetchBytes(ctx, client, url+".sig")
	if err != nil {
		return nil, fmt.Errorf("download signature: %w", err)
	}
	return SaveVerifiedDB(opts.Home, payload, sig, pub)
}

func fetchBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
}
