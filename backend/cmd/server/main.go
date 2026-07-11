package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/auth"
	"github.com/ace-solutions/restaurant-backend/internal/config"
	"github.com/ace-solutions/restaurant-backend/internal/db"
	"github.com/ace-solutions/restaurant-backend/internal/expense"
	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"github.com/ace-solutions/restaurant-backend/internal/menu"
	"github.com/ace-solutions/restaurant-backend/internal/mqttx"
	"github.com/ace-solutions/restaurant-backend/internal/order"
	"github.com/ace-solutions/restaurant-backend/internal/party"
	"github.com/ace-solutions/restaurant-backend/internal/storage"
	"github.com/ace-solutions/restaurant-backend/internal/tables"
	"github.com/ace-solutions/restaurant-backend/internal/waiters"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

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

	mongoX, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("mongo connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoX.Close(shutdownCtx)
	}()
	slog.Info("mongo connected", "db", cfg.MongoDB)

	if err := mongoX.EnsureIndexes(ctx); err != nil {
		slog.Error("ensure indexes failed", "err", err)
		os.Exit(1)
	}

	var mqttClient *mqttx.Client
	if cfg.MQTTBroker != "" {
		mqttClient, err = mqttx.Connect(mqttx.Config{
			Broker:   cfg.MQTTBroker,
			ClientID: cfg.MQTTClientID,
			Username: cfg.MQTTUsername,
			Password: cfg.MQTTPassword,
		})
		if err != nil {
			slog.Error("mqtt connect failed", "err", err)
			os.Exit(1)
		}
		defer mqttClient.Disconnect()
		slog.Info("mqtt connected", "broker", cfg.MQTTBroker)
	} else {
		slog.Warn("MQTT_BROKER not set — realtime features disabled")
	}

	restaurantID, err := bson.ObjectIDFromHex(cfg.DefaultRestaurantID)
	if err != nil {
		slog.Error("invalid DEFAULT_RESTAURANT_ID", "err", err)
		os.Exit(1)
	}

	istanbul, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		slog.Error("load timezone failed", "err", err)
		os.Exit(1)
	}

	authSvc := auth.New(mongoX.DB, restaurantID, istanbul)
	cookieSecure := cfg.Env != "dev"
	authH := auth.NewHandler(authSvc, cookieSecure, cfg.CookieDomain)

	var imageStore *storage.S3
	if cfg.S3Bucket != "" {
		imageStore, err = storage.New(ctx, cfg.S3Bucket, cfg.S3Region, cfg.S3PublicBaseURL)
		if err != nil {
			slog.Error("s3 init failed", "err", err)
			os.Exit(1)
		}
		slog.Info("s3 image upload enabled", "bucket", cfg.S3Bucket, "region", cfg.S3Region)
	} else {
		slog.Warn("S3_BUCKET not set — menu image upload disabled")
	}

	menuSvc := menu.New(mongoX.DB)
	menuH, err := menu.NewHandler(menuSvc, cfg.DefaultRestaurantID, imageStore)
	if err != nil {
		slog.Error("menu handler init failed", "err", err)
		os.Exit(1)
	}

	tablesSvc := tables.New(mongoX.DB)
	tablesH, err := tables.NewHandler(tablesSvc, cfg.DefaultRestaurantID)
	if err != nil {
		slog.Error("tables handler init failed", "err", err)
		os.Exit(1)
	}

	waitersSvc := waiters.New(mongoX.DB)
	waitersH, err := waiters.NewHandler(waitersSvc, cfg.DefaultRestaurantID)
	if err != nil {
		slog.Error("waiters handler init failed", "err", err)
		os.Exit(1)
	}

	partySvc := party.New(mongoX.DB)
	partyH, err := party.NewHandler(partySvc, cfg.DefaultRestaurantID)
	if err != nil {
		slog.Error("party handler init failed", "err", err)
		os.Exit(1)
	}

	expenseSvc := expense.New(mongoX.DB, istanbul, partySvc)
	expenseH, err := expense.NewHandler(expenseSvc, cfg.DefaultRestaurantID, istanbul)
	if err != nil {
		slog.Error("expense handler init failed", "err", err)
		os.Exit(1)
	}

	orderSvc := order.New(mongoX.DB, mqttClient)
	orderH, err := order.NewHandler(orderSvc, cfg.DefaultRestaurantID)
	if err != nil {
		slog.Error("order handler init failed", "err", err)
		os.Exit(1)
	}

	r := httpx.NewRouter(cfg.CORSAllowedOrigins)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := mongoX.Client.Ping(pingCtx, nil); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints (login, logout, me, qr exchange).
		authH.MountPublic(r)
		// Public read-only menu for the QR menu (qr.gunguzelbahce.online).
		menuH.MountPublic(r)

		// Either admin or waiter session can read these — shared data.
		r.Group(func(r chi.Router) {
			r.Use(authSvc.RequireAuthenticated)
			menuH.Mount(r)
			tablesH.MountReadOnly(r)
			orderH.Mount(r)
		})

		// Admin only: configuration + QR issuance.
		r.Group(func(r chi.Router) {
			r.Use(authSvc.RequireAdmin)
			menuH.MountAdmin(r)
			orderH.MountAdmin(r)
			tablesH.MountAdmin(r)
			waitersH.MountAdmin(r)
			partyH.MountAdmin(r)
			expenseH.MountAdmin(r)
			authH.MountAdmin(r)
		})
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "err", err)
	}
}
