package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config tüm ayarları ortam değişkenlerinden okur; gizli değerler kodda veya repoda tutulmaz.
type Config struct {
	Port           string
	DBPath         string
	JWTSecret      []byte
	EncryptionKey  []byte // 32 bayt (AES-256)
	AllowedOrigins []string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Port:   getenv("PORT", "8080"),
		DBPath: getenv("DB_PATH", "data/fullstackx.db"),
	}

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return cfg, errors.New("JWT_SECRET en az 32 karakter olmalı")
	}
	cfg.JWTSecret = []byte(secret)

	key, err := base64.StdEncoding.DecodeString(os.Getenv("ENCRYPTION_KEY"))
	if err != nil || len(key) != 32 {
		return cfg, errors.New("ENCRYPTION_KEY, 32 baytlık anahtarın base64 hali olmalı (örn. `openssl rand -base64 32`)")
	}
	cfg.EncryptionKey = key

	for _, o := range strings.Split(getenv("ALLOWED_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	if len(cfg.AllowedOrigins) == 0 {
		return cfg, fmt.Errorf("ALLOWED_ORIGINS boş olamaz")
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
