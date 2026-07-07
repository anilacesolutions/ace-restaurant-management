package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr            string
	MongoURI            string
	MongoDB             string
	MQTTBroker          string
	MQTTClientID        string
	MQTTUsername        string
	MQTTPassword        string
	Env                 string
	DefaultRestaurantID string
	CORSAllowedOrigins  []string

	// CookieDomain scopes session cookies to a parent domain in prod
	// (e.g. ".gunguzelbahce.online") so app + API subdomains share them.
	// Empty in dev = host-only cookies.
	CookieDomain string

	// S3 — menu item images. Optional: when S3Bucket is empty, image upload is
	// disabled and the presign endpoint returns 501. Everything else works.
	// Credentials come from the default AWS chain (env / ~/.aws / IAM role).
	S3Bucket        string
	S3Region        string
	S3PublicBaseURL string // e.g. a CloudFront domain; empty = derive from bucket+region
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		MongoURI:            os.Getenv("MONGO_URI"),
		MongoDB:             getEnv("MONGO_DB", "restaurant"),
		MQTTBroker:          os.Getenv("MQTT_BROKER"),
		MQTTClientID:        getEnv("MQTT_CLIENT_ID", "restaurant-backend"),
		MQTTUsername:        os.Getenv("MQTT_USERNAME"),
		MQTTPassword:        os.Getenv("MQTT_PASSWORD"),
		Env:                 getEnv("APP_ENV", "dev"),
		DefaultRestaurantID: os.Getenv("DEFAULT_RESTAURANT_ID"),
		CORSAllowedOrigins:  splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		CookieDomain:        os.Getenv("COOKIE_DOMAIN"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
		S3Region:            getEnv("AWS_REGION", "eu-west-1"),
		S3PublicBaseURL:     os.Getenv("S3_PUBLIC_BASE_URL"),
	}

	if cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}
	return cfg, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			seg := s[start:i]
			for len(seg) > 0 && seg[0] == ' ' {
				seg = seg[1:]
			}
			for len(seg) > 0 && seg[len(seg)-1] == ' ' {
				seg = seg[:len(seg)-1]
			}
			if seg != "" {
				out = append(out, seg)
			}
			start = i + 1
		}
	}
	return out
}

func getEnv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(k string, fallback int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
