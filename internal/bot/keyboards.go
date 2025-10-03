package bot

import (
	"GreenAssistantBot/internal/database/models"
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
			tgbotapi.NewKeyboardButton("📒 Заметки"),
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
			tgbotapi.NewKeyboardButton("🌡️ Уведомления о погоде"),
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

	// ЗАКОММЕНТИРОВАТЬ ЭТУ СТРОКУ:
	// storage.ClearUserData(chatID)
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

	//storage.ClearUserData(chatID)
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

// CreateNotesMenuKeyboard создает клавиатуру для меню заметок
func CreateNotesMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Новая заметка"),
			tgbotapi.NewKeyboardButton("📁 Мои заметки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛠️ Управление заметками"),
			tgbotapi.NewKeyboardButton("📂 Управление категориями"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
		),
	)
}

// CreateCategoriesKeyboard создает клавиатуру с категориями пользователя
func CreateCategoriesKeyboard(categories []models.Category) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard()

	// Добавляем категории по 2 в ряд
	for i := 0; i < len(categories); i += 2 {
		row := tgbotapi.NewKeyboardButtonRow()
		row = append(row, tgbotapi.NewKeyboardButton(categories[i].Name))

		if i+1 < len(categories) {
			row = append(row, tgbotapi.NewKeyboardButton(categories[i+1].Name))
		}

		keyboard.Keyboard = append(keyboard.Keyboard, row)
	}

	// Добавляем кнопку возврата
	keyboard.Keyboard = append(keyboard.Keyboard,
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Новая категория"),
			tgbotapi.NewKeyboardButton("⬅️ Назад к заметкам"),
		),
	)

	return keyboard
}

// CreateCategoriesManagementKeyboard создает клавиатуру для управления категориями
func CreateCategoriesManagementKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Создать категорию"),
			tgbotapi.NewKeyboardButton("✏️ Редактировать категории"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🗑️ Удалить категорию"),
			tgbotapi.NewKeyboardButton("⬅️ Назад к заметкам"),
		),
	)
}

// CreateBackKeyboard создает простую клавиатуру с кнопкой назад
func CreateBackKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
		),
	)
}

func CreateNotesViewKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Новая заметка"),
			tgbotapi.NewKeyboardButton("📸 Медиа-заметки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛠️ Управление заметками"),
			tgbotapi.NewKeyboardButton("⬅️ Назад к заметкам"),
		),
	)
}

// CreateConfirmationKeyboard создает клавиатуру для подтверждения действий
func CreateConfirmationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Да"),
			tgbotapi.NewKeyboardButton("❌ Нет"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
		),
	)
}

// CreateNotesManagementKeyboard создает клавиатуру для управления заметками
func CreateNotesManagementKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✏️ Редактировать заметку"),
			tgbotapi.NewKeyboardButton("🗑️ Удалить заметку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад к заметкам"),
		),
	)
}

// CreateNoteActionsKeyboard создает клавиатуру действий для конкретной заметки
func CreateNoteActionsKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✏️ Редактировать"),
			tgbotapi.NewKeyboardButton("🗑️ Удалить"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад к списку"),
		),
	)
}

// CreateNoteEditKeyboard создает клавиатуру для редактирования заметки
func CreateNoteEditKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Редактировать текст"),
			tgbotapi.NewKeyboardButton("📂 Изменить категорию"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
		),
	)
}

func (h *NotesHandler) sendMessageWithoutKeyboard(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message without keyboard: %v", err)
	}
}
