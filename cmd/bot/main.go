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

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
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
