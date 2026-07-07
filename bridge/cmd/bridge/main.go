package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ace-solutions/restaurant-bridge/internal/config"
	"github.com/ace-solutions/restaurant-bridge/internal/mqttx"
	"github.com/ace-solutions/restaurant-bridge/internal/printer"
)

// kitchenPrintMsg is the wire format the backend publishes on
// restaurant/{id}/kitchen/print. Keep in sync with backend.
type kitchenPrintMsg struct {
	OrderID string `json:"orderId"`
	Header  string `json:"header"`
	Footer  string `json:"footer,omitempty"`
	Items   []struct {
		Qty  int    `json:"qty"`
		Name string `json:"name"`
		Note string `json:"note,omitempty"`
	} `json:"items"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	prn := printer.New(printer.Mode(cfg.PrinterMode), cfg.PrinterAddr, cfg.PrinterCols)

	mq, err := mqttx.Connect(mqttx.Config{
		Broker:   cfg.MQTTBroker,
		ClientID: cfg.MQTTClientID,
		Username: cfg.MQTTUsername,
		Password: cfg.MQTTPassword,
	})
	if err != nil {
		slog.Error("mqtt connect failed", "err", err)
		os.Exit(1)
	}
	defer mq.Disconnect()

	topic := fmt.Sprintf("restaurant/%s/kitchen/print", cfg.RestaurantID)
	err = mq.Subscribe(topic, 1, func(_ string, payload []byte) {
		var msg kitchenPrintMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			slog.Error("bad print payload", "err", err)
			return
		}
		ticket := printer.Ticket{
			Header:    msg.Header,
			OrderID:   msg.OrderID,
			Footer:    msg.Footer,
			PrintedAt: time.Now(),
		}
		for _, it := range msg.Items {
			ticket.Items = append(ticket.Items, printer.LineItem{
				Qty:  it.Qty,
				Name: it.Name,
				Note: it.Note,
			})
		}
		if err := prn.Print(ticket); err != nil {
			slog.Error("print failed", "orderId", msg.OrderID, "err", err)
			return
		}
		slog.Info("printed", "orderId", msg.OrderID, "items", len(msg.Items))
	})
	if err != nil {
		slog.Error("subscribe failed", "topic", topic, "err", err)
		os.Exit(1)
	}
	slog.Info("bridge ready", "restaurant", cfg.RestaurantID, "topic", topic, "printer", cfg.PrinterMode)

	<-ctx.Done()
	slog.Info("shutting down")
}
