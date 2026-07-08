package tui

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type SavedConfig struct {
	Profile     int    `json:"profile"`
	InputMode   int    `json:"input_mode"`
	Port        string `json:"port"`
	Threads     string `json:"threads"`
	Timeout     string `json:"timeout"`
	MaxTargets  string `json:"max_targets"`
	EnableIPv6  bool   `json:"enable_ipv6"`
	Verbose     bool   `json:"verbose"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "reality-tui")
}

func configPath() string {
	return filepath.Join(configDir(), "settings.json")
}

func LoadConfig() *SavedConfig {
	cfg := &SavedConfig{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

func SaveConfig(cfg *SavedConfig) {
	os.MkdirAll(configDir(), 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0644)
}

func generateX25519Keys() (string, string, error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("key generation failed: %w", err)
	}
	return fmt.Sprintf("%x", priv.Bytes()), fmt.Sprintf("%x", priv.PublicKey().Bytes()), nil
}

func GenerateRealityConfig(serverName string) string {
	priv, pub, err := generateX25519Keys()
	if err != nil {
		return fmt.Sprintf("// 生成密钥失败: %v", err)
	}
	shortID := "6ba85179e30d4fc2"
	return fmt.Sprintf(`{
  "serverName": "%s",
  "fingerprint": "chrome",
  "publicKey": "%s",
  "privateKey": "%s",
  "shortIds": ["%s"]
}`, serverName, pub, priv, shortID)
}
