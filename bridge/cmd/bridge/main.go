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
	"github.com/joho/godotenv"
)

// kitchenPrintMsg is the wire format the backend publishes on
// restaurant/{id}/kitchen/print. Keep in sync with backend.
type kitchenPrintMsg struct {
	OrderID     string `json:"orderId"`
	TableNumber int    `json:"tableNumber"`
	Header      string `json:"header"` // optional override; else built from table
	Footer      string `json:"footer,omitempty"`
	Items       []struct {
		Qty  int    `json:"qty"`
		Name string `json:"name"`
		Note string `json:"note,omitempty"`
	} `json:"items"`
}

// adisyonPrintMsg is the wire format the backend publishes on
// restaurant/{id}/cashier/print (customer bill). Money is integer kuruş.
type adisyonPrintMsg struct {
	TableNumber int `json:"tableNumber"`
	Items       []struct {
		Qty       int    `json:"qty"`
		Name      string `json:"name"`
		LineTotal int64  `json:"lineTotal"`
		Note      string `json:"note,omitempty"`
	} `json:"items"`
	KDV        map[string]int64 `json:"kdv"`
	GrandTotal int64            `json:"grandTotal"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load a .env sitting next to the binary if present (Windows convenience —
	// the operator just drops a .env beside the .exe). Real env vars still win.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Two printers, split by role. Kitchen ticket has no logo; adisyon does.
	kitchenPrn := printer.New(printer.Mode(cfg.PrinterMode), cfg.KitchenAddr, cfg.KitchenCols)
	kitchenPrn.SetLogo(false)
	adisyonPrn := printer.New(printer.Mode(cfg.PrinterMode), cfg.AdisyonAddr, cfg.AdisyonCols)
	adisyonPrn.SetLogo(cfg.PrinterLogo)

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
		header := msg.Header
		if header == "" {
			header = fmt.Sprintf("MUTFAK - MASA %d", msg.TableNumber)
		}
		ticket := printer.Ticket{
			Header:    header,
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
		if err := kitchenPrn.Print(ticket); err != nil {
			slog.Error("print failed", "orderId", msg.OrderID, "err", err)
			return
		}
		slog.Info("printed", "orderId", msg.OrderID, "items", len(msg.Items))
	})
	if err != nil {
		slog.Error("subscribe failed", "topic", topic, "err", err)
		os.Exit(1)
	}

	// Customer adisyon (receipt) — the cashier's "Adisyon Bas".
	adisyonTopic := fmt.Sprintf("restaurant/%s/cashier/print", cfg.RestaurantID)
	err = mq.Subscribe(adisyonTopic, 1, func(_ string, payload []byte) {
		var msg adisyonPrintMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			slog.Error("bad adisyon payload", "err", err)
			return
		}
		receipt := printer.Receipt{
			RestaurantName: cfg.RestaurantName,
			TableNumber:    msg.TableNumber,
			KDV:            msg.KDV,
			GrandTotal:     msg.GrandTotal,
			PrintedAt:      time.Now(),
		}
		for _, it := range msg.Items {
			receipt.Items = append(receipt.Items, printer.ReceiptLine{
				Qty:       it.Qty,
				Name:      it.Name,
				LineTotal: it.LineTotal,
				Note:      it.Note,
			})
		}
		if err := adisyonPrn.PrintReceipt(receipt); err != nil {
			slog.Error("adisyon print failed", "table", msg.TableNumber, "err", err)
			return
		}
		slog.Info("adisyon printed", "table", msg.TableNumber, "items", len(msg.Items))
	})
	if err != nil {
		slog.Error("subscribe failed", "topic", adisyonTopic, "err", err)
		os.Exit(1)
	}

	slog.Info("bridge ready", "restaurant", cfg.RestaurantID, "mode", cfg.PrinterMode,
		"kitchenPrinter", cfg.KitchenAddr, "kitchenCols", cfg.KitchenCols,
		"adisyonPrinter", cfg.AdisyonAddr, "adisyonCols", cfg.AdisyonCols)

	<-ctx.Done()
	slog.Info("shutting down")
}
