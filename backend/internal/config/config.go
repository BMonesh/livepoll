package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port, Origin, MongoURI, Database, RedisURL, StaticDir string
	Secure                                                bool
	TrustedProxies                                        []string
}

func value(name, fallback string) string {
	if result := os.Getenv(name); result != "" {
		return result
	}
	return fallback
}

func Load() (Config, error) {
	cfg := Config{
		Port:      value("PORT", "8080"),
		Origin:    strings.TrimRight(value("PUBLIC_ORIGIN", "http://localhost:8080"), "/"),
		MongoURI:  value("MONGO_URI", "mongodb://localhost:27017/?replicaSet=rs0&directConnection=true"),
		Database:  value("MONGO_DATABASE", "livepoll"),
		RedisURL:  value("REDIS_URL", "redis://localhost:6379"),
		StaticDir: os.Getenv("STATIC_DIR"),
	}
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, proxy := range strings.Split(raw, ",") {
			cfg.TrustedProxies = append(cfg.TrustedProxies, strings.TrimSpace(proxy))
		}
	}
	origin, err := url.Parse(cfg.Origin)
	if err != nil || origin.Host == "" || origin.Path != "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return cfg, fmt.Errorf("PUBLIC_ORIGIN must be an HTTP(S) origin")
	}
	cfg.Secure = origin.Scheme == "https"
	if !cfg.Secure && origin.Hostname() != "localhost" && origin.Hostname() != "127.0.0.1" {
		return cfg, fmt.Errorf("PUBLIC_ORIGIN must use HTTPS outside localhost")
	}
	return cfg, nil
}
