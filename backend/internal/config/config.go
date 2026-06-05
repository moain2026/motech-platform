// Package config loads runtime configuration strictly from environment
// variables, keeping the system portable (DATABASE_URL / NETBIRD_API_URL
// are the only things that change between dev, Supabase, and self-hosted).
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds all backend runtime settings.
type Config struct {
	DatabaseURL      string // standard PostgreSQL connection string
	NetbirdAPIURL    string // switchable: https://api.netbird.io (cloud) or self-hosted
	NetbirdAPIToken  string // NetBird Personal Access Token (empty => mock mode)
	JWTSecret        string // signing secret for admin/agent JWTs
	MasterKey        string // 32-byte key for AES-256-GCM of stored private keys
	Port             string // HTTP listen port
	SeedAdminEmail   string // first admin seeded if none exists
	SeedAdminPass    string
}

// Load reads a .env file (if present) into the process environment, then
// builds a Config. It fails fast when DATABASE_URL is missing.
func Load() (*Config, error) {
	loadDotEnv(".env")

	c := &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		NetbirdAPIURL:   getOr("NETBIRD_API_URL", "https://api.netbird.io"),
		NetbirdAPIToken: os.Getenv("NETBIRD_API_TOKEN"),
		JWTSecret:       getOr("JWT_SECRET", "insecure-dev-secret"),
		MasterKey:       getOr("MASTER_KEY", "dev-master-key-32bytes-aes256gcm-0123"),
		Port:            getOr("PORT", "8080"),
		SeedAdminEmail:  getOr("SEED_ADMIN_EMAIL", "admin@motech.local"),
		SeedAdminPass:   getOr("SEED_ADMIN_PASSWORD", "admin123"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required (set it in .env or the environment)")
	}
	return c, nil
}

// NetbirdMock reports whether NetBird runs in mock mode (no real token).
func (c *Config) NetbirdMock() bool { return strings.TrimSpace(c.NetbirdAPIToken) == "" }

func getOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadDotEnv loads KEY=VALUE lines from a file into the environment without
// overriding already-set variables. Lines starting with # are ignored.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}
