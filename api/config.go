package api

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var bastilleSpec *BastilleSpecStruct
var configFile = "/usr/local/etc/bastille-api/config.json"
var cfg *ConfigStruct
var APIURL string
var Host string
var Port string

func loadBastilleSpec() (*BastilleSpecStruct, error) {

	logRequest("debug", "loadBastilleSpec", nil, nil, nil)

	var spec BastilleSpecStruct

	data, err := specSheets.ReadFile("bastille.json")
	if err != nil {
		logRequest("error", "Failed to read Bastille spec file", nil, nil, err.Error())
		return nil, err
	}

	if err := json.Unmarshal(data, &spec); err != nil {
		logRequest("error", "Failed to parse Bastille spec", nil, nil, err.Error())
		return nil, err
	}

	bastilleSpec = &spec
	return bastilleSpec, nil
}

func loadConfig() (*ConfigStruct, error) {

	logRequest("debug", "loadConfig", nil, nil, nil)

	data, err := os.ReadFile(configFile)
	if err != nil {
		logRequest("error", "Failed to read config file", nil, nil, err.Error())
		return nil, err
	}

	var c ConfigStruct
	if err := json.Unmarshal(data, &c); err != nil {
		logRequest("error", "Failed to parse config file", nil, nil, err.Error())
		return nil, err
	}

	if c.APIKeys == nil {
		c.APIKeys = make(map[string]APIKeyStruct)
	}

	cfg = &c
	Host = c.Host
	Port = c.Port
	return cfg, nil
}

// bootstrapFirstKey generates an admin-capable API key, persists it, and logs
// the plaintext secret exactly once so a fresh install has a way in. It is only
// called when no keys exist.
func bootstrapFirstKey() error {
	id, secret, entry, err := generateBootstrapKey()
	if err != nil {
		return err
	}
	cfg.APIKeys[id] = entry
	if err := saveConfig(); err != nil {
		return err
	}
	// Intentional one-time disclosure: the plaintext secret is never stored and
	// cannot be recovered later.
	logRequest("info", "No API keys configured; generated a bootstrap key with full "+
		"permissions. Store it now — it will not be shown again. "+
		"Authorization-ID: "+id+"  API key: "+secret, nil, nil, nil)
	return nil
}

func saveConfig() error {

	logRequest("debug", "saveConfig", nil, nil, nil)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logRequest("error", "Failed to marshal config for saving", nil, nil, err.Error())
		return err
	}

	// The config holds API key salts and hashes. Keep the directory and file
	// readable only by the owner (0700/0600) to prevent local offline attacks.
	if err := os.MkdirAll(filepath.Dir(configFile), 0700); err != nil {
		logRequest("error", "Failed to create config directory", nil, nil, err.Error())
		return err
	}

	err = os.WriteFile(configFile, data, 0600)
	if err != nil {
		logRequest("error", "Failed to write config file", nil, nil, err.Error())
		return err
	}

	// Tighten permissions even if the file already existed with a looser mode.
	if err := os.Chmod(configFile, 0600); err != nil {
		logRequest("error", "Failed to set config file permissions", nil, nil, err.Error())
		return err
	}

	return nil
}