package api

import "testing"

// hasDefaultCredential must detect the shipped default key so the server can
// refuse to start with it. Guard for finding C3.
func TestHasDefaultCredential(t *testing.T) {
	// The exact salt/hash pair historically shipped in config.json.sample.
	shipped := &ConfigStruct{
		APIKeys: map[string]APIKeyStruct{
			"bastille": {
				Salt: "my-random-salt",
				Hash: "328de2f095e41086c07746ab22592a6c656045d3a8ff7f0ef1d721760677dea5",
			},
		},
	}
	if !hasDefaultCredential(shipped) {
		t.Fatalf("failed to detect the shipped default credential")
	}

	// Same default key value but re-salted must still be detected.
	resalted := &ConfigStruct{
		APIKeys: map[string]APIKeyStruct{
			"bastille": {Salt: "different-salt", Hash: generateHash(defaultAPIKey, "different-salt")},
		},
	}
	if !hasDefaultCredential(resalted) {
		t.Fatalf("failed to detect the default key under a different salt")
	}

	// A genuine key must not trip the detector.
	safe := &ConfigStruct{
		APIKeys: map[string]APIKeyStruct{
			"op": {Salt: "s", Hash: generateHash("a-real-secret", "s")},
		},
	}
	if hasDefaultCredential(safe) {
		t.Fatalf("false positive on a non-default key")
	}

	if hasDefaultCredential(&ConfigStruct{APIKeys: map[string]APIKeyStruct{}}) {
		t.Fatalf("false positive on empty config")
	}
	if hasDefaultCredential(nil) {
		t.Fatalf("nil config should report no default credential")
	}
}

// generateBootstrapKey must produce a usable, high-entropy, admin-capable key
// whose stored hash verifies against the returned secret.
func TestGenerateBootstrapKey(t *testing.T) {
	id, secret, entry, err := generateBootstrapKey()
	if err != nil {
		t.Fatalf("generateBootstrapKey() error: %v", err)
	}
	if id == "" || secret == "" {
		t.Fatalf("empty id or secret: id=%q secret=%q", id, secret)
	}
	if len(secret) != 64 { // 32 random bytes, hex-encoded
		t.Fatalf("secret entropy too low: len=%d, want 64 hex chars", len(secret))
	}
	if generateHash(secret, entry.Salt) != entry.Hash {
		t.Fatalf("stored hash does not verify against the returned secret")
	}
	if len(entry.Permissions.Bastille) != 1 || entry.Permissions.Bastille[0] != "*" {
		t.Fatalf("bootstrap key missing bastille wildcard permission")
	}
	if len(entry.Permissions.Admin) != 1 || entry.Permissions.Admin[0] != "*" {
		t.Fatalf("bootstrap key missing admin wildcard permission")
	}

	// The generated key must not itself look like the default credential.
	if hasDefaultCredential(&ConfigStruct{APIKeys: map[string]APIKeyStruct{id: entry}}) {
		t.Fatalf("bootstrap key was flagged as the default credential")
	}

	// Two invocations must not collide.
	_, secret2, _, err := generateBootstrapKey()
	if err != nil {
		t.Fatal(err)
	}
	if secret == secret2 {
		t.Fatalf("generateBootstrapKey produced identical secrets")
	}
}
