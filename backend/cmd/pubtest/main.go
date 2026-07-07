// pubtest sends a single kitchen ticket to the printer bridge and exits.
// Handy for verifying the realtime path without driving the UI.
//
//	go run ./cmd/pubtest                 # default sample ticket, masa 7
//	go run ./cmd/pubtest -masa 13        # different table
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ace-solutions/restaurant-backend/internal/config"
	"github.com/ace-solutions/restaurant-backend/internal/mqttx"
)

type item struct {
	Qty  int    `json:"qty"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
}

type msg struct {
	OrderID string `json:"orderId"`
	Header  string `json:"header"`
	Footer  string `json:"footer,omitempty"`
	Items   []item `json:"items"`
}

func main() {
	masa := flag.Int("masa", 7, "table number for the test header")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	if cfg.DefaultRestaurantID == "" {
		slog.Error("DEFAULT_RESTAURANT_ID not set")
		os.Exit(1)
	}

	mq, err := mqttx.Connect(mqttx.Config{
		Broker:   cfg.MQTTBroker,
		ClientID: cfg.MQTTClientID + "-pubtest",
		Username: cfg.MQTTUsername,
		Password: cfg.MQTTPassword,
	})
	if err != nil {
		slog.Error("mqtt connect failed", "err", err)
		os.Exit(1)
	}
	defer mq.Disconnect()

	payload := msg{
		OrderID: "test-" + fmt.Sprintf("%d", *masa),
		Header:  fmt.Sprintf("Masa %d - Garson: Ali", *masa),
		Footer:  "TEST FIS",
		Items: []item{
			{Qty: 2, Name: "Adana Kebap", Note: "az aci"},
			{Qty: 1, Name: "Coban Salata"},
			{Qty: 3, Name: "Ayran"},
			{Qty: 1, Name: "Kunefe", Note: "fistikli"},
		},
	}
	body, _ := json.Marshal(payload)

	topic := mqttx.KitchenPrint(cfg.DefaultRestaurantID)
	if err := mq.Publish(topic, mqttx.QoSExactlyOnce, false, body); err != nil {
		slog.Error("publish failed", "err", err)
		os.Exit(1)
	}
	slog.Info("published", "topic", topic, "bytes", len(body))
}
