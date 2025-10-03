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
	"strings"
	"time"

	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// BotMode представляет режим работы бота
type BotMode string

const (
	WebhookMode BotMode = "webhook"
	PollingMode BotMode = "polling"
)

// Config представляет конфигурацию бота
type Config struct {
	Token      string
	WebhookURL string
	HTTPPort   string
	Mode       BotMode
}

// Функция для поиска и загрузки .env файла
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

// setupWebhook настраивает вебхук для бота
func setupWebhook(api *tgbotapi.BotAPI, webhookURL string) error {
	log.Println("Setting up webhook...")

	webhook, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %v", err)
	}

	_, err = api.Request(webhook)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %v", err)
	}

	info, err := api.GetWebhookInfo()
	if err != nil {
		return fmt.Errorf("failed to get webhook info: %v", err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram bot webhook error: %v", info.LastErrorMessage)
	}

	log.Printf("Webhook configured successfully: %s", info.URL)
	return nil
}

// setupPolling настраивает long polling для бота
func setupPolling(api *tgbotapi.BotAPI) tgbotapi.UpdatesChannel {
	log.Println("Setting up long polling...")

	// Сначала удаляем вебхук, если он был установлен
	_, err := api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		log.Printf("Warning: failed to delete webhook: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := api.GetUpdatesChan(u)
	log.Println("Long polling configured successfully")

	return updates
}

// getBotMode определяет режим работы бота
func getBotMode() BotMode {
	mode := strings.ToLower(os.Getenv("BOT_MODE"))

	switch mode {
	case "webhook":
		return WebhookMode
	case "polling":
		return PollingMode
	default:
		// По умолчанию используем polling для разработки и webhook для продакшена
		if os.Getenv("APP_ENV") == "production" {
			log.Println("Using webhook mode as default for production")
			return WebhookMode
		}
		log.Println("Using polling mode as default for development")
		return PollingMode
	}
}

// loadConfig загружает конфигурацию бота
func loadConfig() *Config {
	return &Config{
		Token:      os.Getenv("BOT_TOKEN"),
		WebhookURL: os.Getenv("BOT_WEBHOOK_URL"),
		HTTPPort:   os.Getenv("HTTP_PORT"),
		Mode:       getBotMode(),
	}
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

	// Загрузка конфигурации
	config := loadConfig()

	if config.Token == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	log.Printf("Starting bot in %s mode", config.Mode)

	// Инициализация базы данных
	db := database.GetConnect()
	_ = db

	// Инициализация хранилища
	botStorage, err := storage.NewMemoryStorage()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	go monitorStorage(botStorage)

	// Инициализация бота
	api, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	// Инициализация обработчиков
	updateHandler := bot.NewUpdateHandler(api, botStorage)
	scheduler := scheduler.NewScheduler(updateHandler.GetMessageHandler())
	scheduler.StartWeatherNotifications()

	var updates tgbotapi.UpdatesChannel
	var server *http.Server

	// Настройка режима работы
	switch config.Mode {
	case WebhookMode:
		if config.WebhookURL == "" {
			log.Fatal("BOT_WEBHOOK_URL is required for webhook mode")
		}
		if config.HTTPPort == "" {
			log.Fatal("HTTP_PORT is required for webhook mode")
		}

		// Настройка вебхука
		err = setupWebhook(api, config.WebhookURL)
		if err != nil {
			log.Fatalf("Failed to setup webhook: %v", err)
		}

		// Запуск HTTP сервера для вебхуков
		server = &http.Server{Addr: ":" + config.HTTPPort}

		go func() {
			log.Printf("Webhook server listening on port %s", config.HTTPPort)
			err := server.ListenAndServe()
			if err != nil && errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Server error: %v", err)
			}
		}()

		updates = api.ListenForWebhook("/")

	case PollingMode:
		// Настройка long polling
		updates = setupPolling(api)

		// Для polling mode также запускаем простой HTTP сервер для health checks
		if config.HTTPPort != "" {
			server = &http.Server{Addr: ":" + config.HTTPPort}

			http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			go func() {
				log.Printf("Health check server listening on port %s", config.HTTPPort)
				err := server.ListenAndServe()
				if err != nil && errors.Is(err, http.ErrServerClosed) {
					log.Printf("Health check server error: %v", err)
				}
			}()
		}

	default:
		log.Fatalf("Unknown bot mode: %s", config.Mode)
	}

	// Запуск обработки обновлений
	go updateHandler.HandleUpdates(updates)

	// Миграции базы данных
	err = database.AutoMigrate()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is running...")

	// Ожидание сигнала завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("Shutting down bot...")

	// Graceful shutdown
	if config.Mode == WebhookMode || (config.Mode == PollingMode && config.HTTPPort != "") {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Удаляем вебхук при завершении
		_, err := api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
		if err != nil {
			log.Printf("Error deleting webhook: %v", err)
		}

		// Останавливаем HTTP сервер, если он был запущен
		if server != nil {
			err = server.Shutdown(ctx)
			if err != nil {
				log.Printf("HTTP server Shutdown: %v", err)
			}
		}
	}

	log.Println("Bot gracefully stopped")
}
