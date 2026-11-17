package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"livepoll/internal/auth"
	"livepoll/internal/config"
	"livepoll/internal/polls"
	"livepoll/internal/realtime"
	"livepoll/internal/server"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI).SetTimeout(6 * time.Second))
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(closeCtx)
	}()
	if err := mongoClient.Ping(startup, nil); err != nil {
		return err
	}
	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return err
	}
	redisOptions.ContextTimeoutEnabled = true
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	if err := redisClient.Ping(startup).Err(); err != nil {
		return err
	}

	db := mongoClient.Database(cfg.Database)
	accounts, pollStore := auth.NewStore(db), polls.NewStore(db)
	if err := accounts.Index(startup); err != nil {
		return err
	}
	if err := pollStore.Index(startup); err != nil {
		return err
	}
	hub := realtime.New(redisClient)
	go hub.Run(ctx)
	router, err := server.Router(cfg, accounts, pollStore, hub, redisClient, mongoClient)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr: ":" + cfg.Port, Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, IdleTimeout: 75 * time.Second, MaxHeaderBytes: 32 << 10,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	result := make(chan error, 1)
	go func() {
		slog.Info("polling app started", "origin", cfg.Origin, "port", cfg.Port)
		result <- httpServer.ListenAndServe()
	}()
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdown)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("polling app stopped", "error", err)
		os.Exit(1)
	}
}
