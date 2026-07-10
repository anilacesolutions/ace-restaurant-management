package config

import (
	"fmt"
	"os"
)

type Config struct {
	RestaurantID   string
	RestaurantName string // printed as the adisyon header (e.g. "GUN GUZELBAHCE")
	MQTTBroker     string
	MQTTClientID   string
	MQTTUsername   string
	MQTTPassword   string

	PrinterMode string // "windows", "usb", "network", "stdout" (dev)
	PrinterAddr string // windows: printer name; network: host:port; usb: device path
	PrinterCols int    // characters per line (58mm=32, 80mm=48)
	PrinterLogo bool   // print the logo bitmap atop the adisyon (default true)
}

func Load() (*Config, error) {
	cfg := &Config{
		RestaurantID:   os.Getenv("RESTAURANT_ID"),
		RestaurantName: getEnv("RESTAURANT_NAME", "GUN GUZELBAHCE"),
		MQTTBroker:     os.Getenv("MQTT_BROKER"),
		MQTTClientID:   getEnv("MQTT_CLIENT_ID", "restaurant-bridge"),
		MQTTUsername:   os.Getenv("MQTT_USERNAME"),
		MQTTPassword:   os.Getenv("MQTT_PASSWORD"),
		PrinterMode:    getEnv("PRINTER_MODE", "stdout"),
		PrinterAddr:    os.Getenv("PRINTER_ADDR"),
		PrinterCols:    getEnvInt("PRINTER_COLS", 32),
		PrinterLogo:    getEnv("PRINTER_LOGO", "true") != "false",
	}
	if cfg.RestaurantID == "" {
		return nil, fmt.Errorf("RESTAURANT_ID is required")
	}
	if cfg.MQTTBroker == "" {
		return nil, fmt.Errorf("MQTT_BROKER is required")
	}
	return cfg, nil
}

func getEnv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(k string, fallback int) int {
	if v := os.Getenv(k); v != "" {
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err == nil {
			return n
		}
	}
	return fallback
}
