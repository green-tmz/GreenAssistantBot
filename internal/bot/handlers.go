package bot

import (
	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/database/models"
	"GreenAssistantBot/internal/storage"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
)

type UpdateHandler struct {
	bot        *tgbotapi.BotAPI
	storage    storage.BotStorage
	msgHandler *MessageHandler
}

func NewUpdateHandler(bot *tgbotapi.BotAPI, storage storage.BotStorage) *UpdateHandler {
	return &UpdateHandler{
		bot:        bot,
		storage:    storage,
		msgHandler: NewMessageHandler(bot, storage),
	}
}

func (h *UpdateHandler) handleUserState(chatID int64, state, userText, userName, lastName string) bool {
	switch state {
	case StateWaitingForName:
		user := &models.User{TelegramID: chatID, FirstName: userText, UserName: userName, LastName: lastName}
		if err := database.SaveOrUpdateUser(user); err != nil {
			log.Printf("Error saving user: %v", err)
			return false
		}
		h.msgHandler.AskForCity(chatID)
		return true

	case StateWaitingForCity:
		user := &models.User{TelegramID: chatID, City: userText}
		if err := database.SaveOrUpdateUser(user); err != nil {
			log.Printf("Error saving user: %v", err)
			return false
		}
		h.msgHandler.CompleteProfile(chatID)
		return true

	case StateChangingNameFromProfile:
		user := &models.User{TelegramID: chatID, FirstName: userText}
		if err := database.SaveOrUpdateUser(user); err != nil {
			log.Printf("Error saving user: %v", err)
			return false
		}
		h.msgHandler.SendProfileSettings(chatID)
		h.storage.SetUserState(chatID, "")
		return true

	case StateChangingCityFromProfile:
		user := &models.User{TelegramID: chatID, City: userText}
		if err := database.SaveOrUpdateUser(user); err != nil {
			log.Printf("Error saving user: %v", err)
			return false
		}
		h.msgHandler.SendProfileSettings(chatID)
		h.storage.SetUserState(chatID, "")
		return true

	case StateWaitingForWeatherCity:
		h.msgHandler.SendWeather(chatID, userText)
		h.storage.SetUserState(chatID, "")
		return true
	}
	return false
}

func (h *UpdateHandler) HandleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil || update.Message.From.IsBot {
			continue
		}

		chatID := update.Message.Chat.ID
		userText := update.Message.Text

		log.Printf("[%d]: %s", chatID, userText)

		adminChatID := os.Getenv("ADMIN_CHAT_ID")
		if adminChatID != "" && adminChatID != fmt.Sprintf("%d", chatID) {
			log.Printf("Chat id: %d", chatID)
			continue
		}

		// Обработка состояний
		if state, exists := h.storage.GetUserState(chatID); exists {
			if h.handleUserState(chatID, state, userText, update.Message.From.UserName, update.Message.From.LastName) {
				continue
			}
		}

		// Обработка команд
		switch userText {
		case "/start":
			h.msgHandler.SendStartMessage(chatID)
			if exists, _ := database.UserExists(chatID); !exists {
				user := &models.User{
					TelegramID: chatID,
					UserName:   update.Message.From.UserName,
					FirstName:  update.Message.From.FirstName,
					LastName:   update.Message.From.LastName,
				}
				database.SaveOrUpdateUser(user)
				h.msgHandler.AskForName(chatID)
			}

		case "✏️ Ваше имя":
			h.msgHandler.AskForName(chatID)
			h.storage.SetUserState(chatID, StateChangingNameFromProfile)

		case "🚩 Ваш город":
			h.msgHandler.AskForCity(chatID)
			h.storage.SetUserState(chatID, StateChangingCityFromProfile)

		case "ℹ️ Информация":
			h.msgHandler.SendInfo(chatID)

		case "⚙️ Настройки":
			h.msgHandler.SendSettingsMenu(chatID)

		case "📞 Поддержка":
			h.msgHandler.SendSupport(chatID)

		case "🔔 Уведомления":
			h.msgHandler.SendNotificationsSettings(chatID)

		case "👤 Профиль":
			h.msgHandler.SendProfileSettings(chatID)

		case "🌡️Погода":
			if user, err := database.GetUserByTelegramID(chatID); err == nil && user.City != "" {
				h.msgHandler.SendWeather(chatID, user.City)
			} else {
				h.msgHandler.sendMessage(chatID, "🌍 Введите название города:", CreateMainMenuKeyboard())
				h.storage.SetUserState(chatID, StateWaitingForWeatherCity)
			}

		case "🌡️ Уведомления о погоде":
			// Получаем текущее состояние уведомлений пользователя
			user, err := database.GetUserByTelegramID(chatID)
			if err != nil {
				log.Printf("Error getting user: %v", err)
				h.msgHandler.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.", CreateSettingsMenuKeyboard())
				continue
			}

			// Изменяем состояние уведомлений
			user.WeatherNotifications = !user.WeatherNotifications
			err = database.SaveOrUpdateUser(user)
			if err != nil {
				log.Printf("Error updating user: %v", err)
				h.msgHandler.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.", CreateSettingsMenuKeyboard())
				continue
			}

			// Отправляем подтверждение
			status := "включены"
			if !user.WeatherNotifications {
				status = "выключены"
			}
			h.msgHandler.sendMessage(chatID, fmt.Sprintf("Уведомления о погоде %s", status), CreateSettingsMenuKeyboard())

		case "⬅️ Назад", "🏠 В начало":
			h.msgHandler.SendMainMenu(chatID)

		default:
			h.msgHandler.sendMessage(chatID, "Используйте меню для навигации", CreateMainMenuKeyboard())
		}
	}
}

func (h *MessageHandler) SendMessage(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	return h.sendMessage(chatID, text, keyboard)
}

func (h *UpdateHandler) GetMessageHandler() *MessageHandler {
	return h.msgHandler
}
