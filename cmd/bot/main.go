package main

import (
	"GreenAssistantBot/internal/bot"
	"GreenAssistantBot/internal/scheduler"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func monitorStorage(storage storage.BotStorage) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if statsStorage, ok := storage.(interface{ GetStats() map[string]interface{} }); ok {
			stats := statsStorage.GetStats()
			log.Printf("Storage stats: %+v", stats)
		}
	}
}

func loadEnv() error {
	// Попробуем несколько возможных путей
	possiblePaths := []string{
		".env",
		"./.env",
		"../.env",
		"../../.env",
	}

	var loaded bool
	for _, path := range possiblePaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from: %s", path)
			loaded = true
			break
		}
	}

	if !loaded {
		// Получаем текущую рабочую директорию
		wd, _ := os.Getwd()
		log.Printf("Current working directory: %s", wd)

		// Покажем какие файлы есть в директории
		files, _ := filepath.Glob("*")
		log.Printf("Files in current directory: %v", files)

		return fmt.Errorf("could not load .env file from any path")
	}

	return nil
}

func main() {
	// Загружаем .env файл
	err := loadEnv()
	if err != nil {
		log.Printf("Warning: %v", err)
		log.Println("Continuing with system environment variables...")
	}

	// Проверим наличие обязательных переменных
	requiredVars := []string{"BOT_TOKEN", "HTTP_PORT", "BOT_WEBHOOK_URL"}
	for _, envVar := range requiredVars {
		if os.Getenv(envVar) == "" {
			log.Printf("Warning: Environment variable %s is not set", envVar)
		}
	}

	db := database.GetConnect()
	_ = db

	// Инициализация хранилища
	botStorage, err := storage.NewMemoryStorage()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	go monitorStorage(botStorage)

	port := os.Getenv("HTTP_PORT")
	server := http.Server{Addr: ":" + port}

	go func() {
		fmt.Println("Listening on port " + port)
		err := server.ListenAndServe()
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Инициализация бота
	api, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	// Настройка webhook
	webhook, err := tgbotapi.NewWebhook(os.Getenv("BOT_WEBHOOK_URL"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = api.Request(webhook)
	if err != nil {
		log.Fatal(err)
	}

	info, err := api.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram bot error data: %v", info.LastErrorMessage)
	}

	updates := api.ListenForWebhook("/")
	updateHandler := bot.NewUpdateHandler(api, botStorage)
	go updateHandler.HandleUpdates(updates)

	scheduler := scheduler.NewScheduler(updateHandler.GetMessageHandler())
	scheduler.StartWeatherNotifications()

	stop = make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	err = database.AutoMigrate()
	if err != nil {
		log.Fatal(err)
	}

	<-stop

	log.Println("Shutting down server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}

	fmt.Println("Server gracefully stopped")
}
