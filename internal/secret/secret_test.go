package secret_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/secret"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func TestValueNeverLeaksRaw(t *testing.T) {
	raw := testutil.FakeSecret(1, "github-pat-classic")
	v := secret.NewValue(raw, 4, 4)

	formats := []string{
		v.String(),
		v.Preview(),
		v.GoString(),
		fmt.Sprintf("%s", v),
		fmt.Sprintf("%v", v),
		fmt.Sprintf("%+v", v),
		fmt.Sprintf("%#v", v),
		fmt.Sprintf("%q", v),
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	formats = append(formats, string(b))

	type wrapper struct {
		Secret secret.Value `json:"secret"`
	}
	wb, err := json.Marshal(wrapper{Secret: v})
	if err != nil {
		t.Fatalf("json.Marshal wrapper: %v", err)
	}
	formats = append(formats, string(wb))

	for i, out := range formats {
		if strings.Contains(out, raw) {
			t.Fatalf("format[%d] leaked raw secret: %q", i, out)
		}
		if !strings.Contains(out, "••••••") {
			t.Fatalf("format[%d] missing mask bullets: %q", i, out)
		}
	}
}

func TestMaskPreviewRules(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		prefixLen  int
		suffixLen  int
		wantMasked string
	}{
		{
			name:       "standard prefix and suffix",
			raw:        "ghp_1234567890abcdefghijklmnopqrstuva9Xz",
			prefixLen:  4,
			suffixLen:  4,
			wantMasked: "ghp_••••••a9Xz",
		},
		{
			name:       "short value masks completely",
			raw:        "short123",
			prefixLen:  4,
			suffixLen:  4,
			wantMasked: "••••••",
		},
		{
			name:       "threshold length 11 masks completely",
			raw:        "12345678901",
			prefixLen:  4,
			suffixLen:  4,
			wantMasked: "••••••",
		},
		{
			name:       "length 12 keeps 4+4 because 4 masked remain",
			raw:        "1234abcd5678",
			prefixLen:  4,
			suffixLen:  4,
			wantMasked: "1234••••••5678",
		},
		{
			name:       "zero prefix and suffix",
			raw:        "supersecretpassword123",
			prefixLen:  0,
			suffixLen:  0,
			wantMasked: "••••••",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := secret.NewValue(tc.raw, tc.prefixLen, tc.suffixLen)
			if got := v.Preview(); got != tc.wantMasked {
				t.Errorf("Preview() = %q, want %q", got, tc.wantMasked)
			}
			if strings.Contains(v.Preview(), tc.raw) {
				t.Errorf("Preview() leaked raw value %q", tc.raw)
			}
		})
	}
}

func TestFingerprintDeterministic(t *testing.T) {
	raw1 := testutil.FakeSecret(1, "npm-token")
	raw2 := testutil.FakeSecret(2, "npm-token")

	v1a := secret.NewValue(raw1, 4, 4)
	v1b := secret.NewValue(raw1, 2, 2) // different preview lengths, same raw secret
	v2 := secret.NewValue(raw2, 4, 4)

	if v1a.Fingerprint() != v1b.Fingerprint() {
		t.Fatalf("same raw secret produced different fingerprints: %q vs %q", v1a.Fingerprint(), v1b.Fingerprint())
	}
	if v1a.Fingerprint() == v2.Fingerprint() {
		t.Fatalf("different raw secrets produced identical fingerprint: %q", v1a.Fingerprint())
	}
	if !strings.HasPrefix(v1a.Fingerprint(), "sha256:") {
		t.Fatalf("Fingerprint() = %q, want sha256: prefix", v1a.Fingerprint())
	}
	if len(v1a.Fingerprint()) != len("sha256:")+64 {
		t.Fatalf("unexpected fingerprint length: %q", v1a.Fingerprint())
	}
}

func TestMatchCarriesNoRawSecret(t *testing.T) {
	raw := testutil.FakeSecret(3, "openai-api-key")
	m := secret.Match{
		Kind:  "openai-api-key",
		Value: secret.NewValue(raw, 5, 4),
		Line:  12,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	s := fmt.Sprintf("%+v %#v %s", m, m, string(b))
	if strings.Contains(s, raw) {
		t.Fatalf("Match leaked raw secret: %s", s)
	}
}
