package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"linkshortener/internal/bot"
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Info("Starting LinkShortener service...", "port", os.Getenv("PORT"))

	godotenv.Load()
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

	db, err := database.ConnectPostgres(postgresURL)
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

	analytics, err := database.ConnectClickHouse(clickhouseAddr, clickhouseUser, clickhousePassword, clickhouseDb)
	if err != nil {
		slog.Error("Could not connect to ClickHouse", "error", err)
		return
	}

	tgBot, err := bot.NewTelegramBot(tgToken, db, cacheDB, analytics)
	if err != nil {
		slog.Error("Could not initialize bot", "error", err)
		return
	}

	go tgBot.Start()

	slog.Info("Service is up and running!")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gracefully...")
}
