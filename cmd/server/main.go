package main

import (
	"context"
	"linkshortener/internal/service"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"linkshortener/internal/bot"
	"linkshortener/internal/cache"
	"linkshortener/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Info("Starting LinkShortener service...", "port", os.Getenv("PORT"))

	err := godotenv.Load()
	if err != nil {
		slog.Warn("Error loading .env file", "error", err)
	}

	clickhouseAddr := os.Getenv("CLICKHOUSE_ADDR")
	clickhouseUser := os.Getenv("CLICKHOUSE_USER")
	clickhousePassword := os.Getenv("CLICKHOUSE_PASSWORD")
	clickhouseDb := os.Getenv("CLICKHOUSE_DB")
	postgresURL := os.Getenv("DB_URL")
	redisUrl := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	tgToken := os.Getenv("TELEGRAM_API_TOKEN")
	port := os.Getenv("PORT")

	if clickhouseAddr == "" ||
		clickhouseUser == "" ||
		clickhousePassword == "" ||
		clickhouseDb == "" ||
		postgresURL == "" ||
		redisUrl == "" ||
		redisPassword == "" ||
		tgToken == "" ||
		port == "" {
		slog.Error("Missing required environment variables")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.ConnectPostgres(ctx, postgresURL)
	if err != nil {
		slog.Error("Could not connect to Postgres", "error", err)
		return
	}
	defer db.Close()

	cacheDB, err := cache.ConnectRedis(redisUrl, redisPassword)
	if err != nil {
		slog.Error("Could not connect to Redis", "error", err)
		return
	}
	defer cacheDB.Close()

	analytics, err := database.ConnectClickHouse(ctx, clickhouseAddr, clickhouseUser, clickhousePassword, clickhouseDb)
	if err != nil {
		slog.Error("Could not connect to ClickHouse", "error", err)
		return
	}
	defer analytics.Close()
	analytics.Start(ctx)

	tgBot, err := bot.NewTelegramBot(tgToken, db, cacheDB, analytics)
	if err != nil {
		slog.Error("Could not initialize bot", "error", err)
		return
	}
	botErr := make(chan error, 1)
	go func() { botErr <- tgBot.Start(ctx) }()

	server := service.NewServer(port, db, cacheDB, analytics)
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Start(ctx) }()

	slog.Info("Service is up and running!")

	select {
	case <-ctx.Done():
		slog.Info("Shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			slog.Error("Server stopped with error", "error", err)
			stop()
		}
	case err := <-botErr:
		if err != nil {
			slog.Error("Bot stopped with error", "error", err)
			stop()
		}
	}

	slog.Info("Shutting down gracefully...")
}
