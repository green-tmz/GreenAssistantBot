package bot

import (
	"GreenAssistantBot/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func CreateMainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("ℹ️ Информация"),
			tgbotapi.NewKeyboardButton("📞 Поддержка"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🌡️Погода"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)
}

func CreateSettingsMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔔 Уведомления"),
			tgbotapi.NewKeyboardButton("👤 Профиль"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
		),
	)
}

func SendSettingsMenu(bot *tgbotapi.BotAPI, chatID int64, storage storage.BotStorage) {
	text := "⚙️ Настройки"
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = CreateSettingsMenuKeyboard()

	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending settings menu: %v", err)
	} else {
		storage.SetLastMessageID(chatID, sentMsg.MessageID)
	}

	storage.ClearUserData(chatID)
}

func SendMainMenu(bot *tgbotapi.BotAPI, chatID int64, storage storage.BotStorage) {
	msg := tgbotapi.NewMessage(chatID, "Выберите один из пунктов")
	msg.ReplyMarkup = CreateMainMenuKeyboard()

	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending main menu: %v", err)
	} else {
		storage.SetLastMessageID(chatID, sentMsg.MessageID)
	}

	storage.ClearUserData(chatID)
}

func CreateProfileMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✏️ Ваше имя"),
			tgbotapi.NewKeyboardButton("🚩 Ваш город"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 В начало"),
		),
	)
}
