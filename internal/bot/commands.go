package bot

import (
	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/storage"
	"GreenAssistantBot/internal/weather"
	"fmt"
	"log"
	_ "strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "gorm.io/gorm"
)

const (
	StateWaitingForCity          = "waiting_for_city"
	StateWaitingForName          = "waiting_for_name"
	StateChangingNameFromProfile = "changing_name_from_profile"
	StateChangingCityFromProfile = "changing_city_from_profile"
	StateWaitingForWeatherCity   = "waiting_for_weather_city"
)

type MessageHandler struct {
	bot     *tgbotapi.BotAPI
	storage storage.BotStorage
}

func NewMessageHandler(bot *tgbotapi.BotAPI, storage storage.BotStorage) *MessageHandler {
	return &MessageHandler{bot: bot, storage: storage}
}

func (h *MessageHandler) sendMessage(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)

	if keyboard.Keyboard != nil {
		msg.ReplyMarkup = keyboard
	} else {
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	}

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	h.storage.SetLastMessageID(chatID, sentMsg.MessageID)
	h.storage.ClearUserData(chatID)
	return nil
}

func (h *MessageHandler) SendStartMessage(chatID int64) {
	text := `👋 Добро пожаловать в GreenAssistantBot!

✨ Основные возможности:
• 👤 Управление профилем
• ⚙️ Настройки
• 🔔 Уведомления
• 📞 Поддержка
• ℹ️ Информация о боте`

	h.sendMessage(chatID, text, CreateMainMenuKeyboard())
}

func (h *MessageHandler) AskForName(chatID int64) {
	h.sendMessage(chatID, "✏️ Пожалуйста, введите ваше имя:", CreateMainMenuKeyboard())
	h.storage.SetUserState(chatID, StateWaitingForName)
}

func (h *MessageHandler) AskForCity(chatID int64) {
	h.sendMessage(chatID, "🚩 Пожалуйста, введите ваш город:", CreateMainMenuKeyboard())
	h.storage.SetUserState(chatID, StateWaitingForCity)
}

func (h *MessageHandler) CompleteProfile(chatID int64) {
	user, err := database.GetUserByTelegramID(chatID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	text := fmt.Sprintf("✅ Анкета заполнена!\n\n👤 Ваш профиль:\n✏️ Имя: %s\n🚩 Город: %s",
		user.FirstName, user.City)

	h.sendMessage(chatID, text, CreateMainMenuKeyboard())
	h.storage.SetUserState(chatID, "")
}

func (h *MessageHandler) SendProfileSettings(chatID int64) {
	user, err := database.GetUserByTelegramID(chatID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	text := fmt.Sprintf("👤 Ваш профиль\n✏️ Имя: %s\n🚩 Город: %s", user.FirstName, user.City)
	h.sendMessage(chatID, text, CreateProfileMenuKeyboard())
}

func (h *MessageHandler) SendWeather(chatID int64, city string) {
	weatherService := weather.NewWeatherService()
	weatherData, err := weatherService.GetWeatherData(city)

	var text string
	if err != nil {
		text = fmt.Sprintf("❌ Не удалось получить данные о погоде для города '%s'", city)
	} else {
		text = weatherService.FormatWeatherMessage(weatherData)
	}

	err = h.sendMessage(chatID, text, CreateMainMenuKeyboard())
	if err != nil {
		return
	}
}

func (h *MessageHandler) DeleteLastBotMessage(chatID int64) {
	if messageID, exists := h.storage.GetLastMessageID(chatID); exists {
		deleteConfig := tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: messageID}
		h.bot.Request(deleteConfig) // Игнорируем ошибку
	}
}
