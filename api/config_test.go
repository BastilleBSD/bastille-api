package api

import (
	"os"
	"path/filepath"
	"testing"
)

// saveConfig must persist the config with owner-only permissions (0600), since
// it contains API key salts and hashes. Guard for finding H2.
func TestSaveConfigPermissions(t *testing.T) {
	InitLogger(false)

	dir := t.TempDir()
	prevConfigFile := configFile
	prevCfg := cfg
	t.Cleanup(func() {
		configFile = prevConfigFile
		cfg = prevCfg
	})

	configFile = filepath.Join(dir, "sub", "config.json")
	cfg = &ConfigStruct{
		Host: "localhost",
		Port: "8888",
		APIKeys: map[string]APIKeyStruct{
			"op": {Salt: "s", Hash: generateHash("secret", "s")},
		},
	}

	if err := saveConfig(); err != nil {
		t.Fatalf("saveConfig() error: %v", err)
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("config file mode = %o, want 0600", perm)
	}

	dirInfo, err := os.Stat(filepath.Dir(configFile))
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Fatalf("config dir mode = %o, want 0700", perm)
	}
}

// Even if the file pre-exists with a loose mode, saveConfig must tighten it.
func TestSaveConfigTightensExistingFile(t *testing.T) {
	InitLogger(false)

	dir := t.TempDir()
	prevConfigFile := configFile
	prevCfg := cfg
	t.Cleanup(func() {
		configFile = prevConfigFile
		cfg = prevCfg
	})

	configFile = filepath.Join(dir, "config.json")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg = &ConfigStruct{APIKeys: map[string]APIKeyStruct{}}

	if err := saveConfig(); err != nil {
		t.Fatalf("saveConfig() error: %v", err)
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("pre-existing config not tightened: mode = %o, want 0600", perm)
	}
}
