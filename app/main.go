package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/remitly-task/stocksim/internal/config"
	"github.com/remitly-task/stocksim/internal/handler"
	"github.com/remitly-task/stocksim/internal/service"
	"github.com/remitly-task/stocksim/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	// Wait for Redis to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		if err := redisClient.Ping(ctx).Err(); err == nil {
			break
		}
		slog.Info("waiting for redis", "addr", cfg.RedisAddr)
		time.Sleep(500 * time.Millisecond)
	}
	slog.Info("connected to redis", "addr", cfg.RedisAddr)

	redisStore, err := store.NewRedisStore(redisClient)
	if err != nil {
		slog.Error("failed to initialize store", "error", err)
		os.Exit(1)
	}

	walletSvc := service.NewWalletService(redisStore)
	bankSvc := service.NewBankService(redisStore)
	logSvc := service.NewLogService(redisStore)

	router := handler.NewRouter(handler.Services{
		Wallet: walletSvc,
		Bank:   bankSvc,
		Log:    logSvc,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCtx.Done()
	slog.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
