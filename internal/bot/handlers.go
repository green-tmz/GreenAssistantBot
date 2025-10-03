package bot

import (
	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/database/models"
	"GreenAssistantBot/internal/storage"
	pmodel "GreenAssistantBot/pkg/models"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UpdateHandler struct {
	bot          *tgbotapi.BotAPI
	storage      storage.BotStorage
	msgHandler   *MessageHandler
	notesHandler *NotesHandler
}

func NewUpdateHandler(bot *tgbotapi.BotAPI, storage storage.BotStorage) *UpdateHandler {
	msgHandler := NewMessageHandler(bot, storage)
	return &UpdateHandler{
		bot:          bot,
		storage:      storage,
		msgHandler:   msgHandler,
		notesHandler: NewNotesHandler(bot, storage, msgHandler),
	}
}

func (h *UpdateHandler) handleUserState(chatID int64, state, userText, userName, lastName string, update tgbotapi.Update) bool {
	log.Printf("handleUserState: chatID=%d, state=%s, userText=%s", chatID, state, userText)

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

	case StateWaitingForCategoryName:
		h.notesHandler.HandleCategoryCreation(chatID, userText)
		return true

	case StateWaitingForNoteCategory:
		// Обработка выбора категории для разных целей
		userData, exists := h.storage.GetUserData(chatID)
		if !exists {
			log.Printf("No user data found for chat %d in state %s", chatID, state)
			h.msgHandler.sendMessage(chatID, "❌ Сессия истекла, начните заново", CreateNotesMenuKeyboard())
			h.storage.SetUserState(chatID, "")
			return true
		}

		purpose := userData.Data
		log.Printf("Category selected: %s for purpose: %s", userText, purpose)

		switch purpose {
		case "new_note":
			// Сохраняем выбранную категорию в отдельном поле
			userData.Category = userText
			h.storage.SetUserData(chatID, userData)
			h.msgHandler.sendMessage(chatID,
				"📝 Отправьте текст, фото, видео, голосовое сообщение или файл для сохранения в заметку:",
				CreateBackKeyboard())
			h.storage.SetUserState(chatID, StateWaitingForNoteContent)
			log.Printf("Waiting for note content for category: %s", userText)

		case "view_notes":
			// Показываем заметки выбранной категории
			h.notesHandler.SendNotesByCategory(chatID, userText)
			h.storage.SetUserState(chatID, "")
		case "delete_category":
			h.notesHandler.HandleDeleteCategory(chatID, userText)
		case "edit_category":
			// Обработка редактирования категории
			h.notesHandler.HandleEditCategory(chatID, userText)
		case "save_forwarded_message":
			// Сохраняем пересланное сообщение в выбранной категории
			h.notesHandler.SaveForwardedMessage(chatID, userText, userData)
		default:
			log.Printf("Unknown purpose: %s", purpose)
			h.msgHandler.sendMessage(chatID, "❌ Неизвестная операция", CreateNotesMenuKeyboard())
			h.storage.SetUserState(chatID, "")
		}
		return true

	case StateDeletingCategory:
		if strings.ToLower(userText) == "да" || userText == "✅ Да" {
			h.notesHandler.ConfirmDeleteCategory(chatID, true)
		} else if strings.ToLower(userText) == "нет" || userText == "❌ Нет" {
			h.notesHandler.ConfirmDeleteCategory(chatID, false)
		} else if userText == "⬅️ Назад" {
			h.msgHandler.sendMessage(chatID, "❌ Удаление отменено", CreateCategoriesManagementKeyboard())
			h.storage.SetUserState(chatID, "")
		} else {
			h.msgHandler.sendMessage(chatID, "❌ Пожалуйста, используйте кнопки для подтверждения", CreateConfirmationKeyboard())
		}
		return true

	case StateEditingCategory:
		h.notesHandler.HandleCategoryUpdate(chatID, userText)
		return true

	case StateWaitingForNoteSelection:
		userData, exists := h.storage.GetUserData(chatID)
		if !exists {
			log.Printf("No user data found for chat %d in state %s", chatID, state)
			h.msgHandler.sendMessage(chatID, "❌ Сессия истекла, начните заново", CreateNotesMenuKeyboard())
			h.storage.SetUserState(chatID, "")
			return true
		}

		purpose := userData.Data
		log.Printf("Note selected: %s for purpose: %s", userText, purpose)

		// Парсим ID заметки из текста (формат: "ID: текст")
		noteIDStr := strings.Split(userText, ":")[0]
		noteID, err := strconv.ParseUint(strings.TrimSpace(noteIDStr), 10, 32)
		if err != nil {
			log.Printf("Error parsing note ID: %v", err)
			h.msgHandler.sendMessage(chatID, "❌ Ошибка при выборе заметки", CreateNotesMenuKeyboard())
			h.storage.SetUserState(chatID, "")
			return true
		}

		switch purpose {
		case "edit_note":
			h.notesHandler.HandleEditNoteSelection(chatID, uint(noteID))
		case "delete_note":
			h.notesHandler.HandleDeleteNoteSelection(chatID, uint(noteID))
		default:
			log.Printf("Unknown purpose: %s", purpose)
			h.msgHandler.sendMessage(chatID, "❌ Неизвестная операция", CreateNotesMenuKeyboard())
			h.storage.SetUserState(chatID, "")
		}
		return true

	case StateEditingNote:
		h.notesHandler.HandleNoteContentUpdate(chatID, userText)
		return true

	case StateDeletingNote:
		if strings.ToLower(userText) == "да" || userText == "✅ Да" {
			h.notesHandler.ConfirmDeleteNote(chatID, true)
		} else if strings.ToLower(userText) == "нет" || userText == "❌ Нет" {
			h.notesHandler.ConfirmDeleteNote(chatID, false)
		} else if userText == "⬅️ Назад" {
			h.msgHandler.sendMessage(chatID, "❌ Удаление отменено", CreateNotesManagementKeyboard())
			h.storage.SetUserState(chatID, "")
		} else {
			h.msgHandler.sendMessage(chatID, "❌ Пожалуйста, используйте кнопки для подтверждения", CreateConfirmationKeyboard())
		}
		return true

	case SaveForwardedMessage:
		// Сохраняем пересланное сообщение в выбранной категории
		userData, _ := h.storage.GetUserData(chatID)
		h.notesHandler.SaveForwardedMessage(chatID, userText, userData)
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

		// Проверяем, является ли сообщение пересланным
		if update.Message.ForwardFrom != nil || update.Message.ForwardFromChat != nil || update.Message.ForwardSenderName != "" {
			log.Printf("Forwarded message: From=%v, FromChat=%v, SenderName=%s",
				update.Message.ForwardFrom,
				update.Message.ForwardFromChat,
				update.Message.ForwardSenderName)
		}

		// Обработка состояний
		if state, exists := h.storage.GetUserState(chatID); exists {
			if h.handleUserState(chatID, state, userText, update.Message.From.UserName, update.Message.From.LastName, update) {
				continue
			}
		}

		// Проверяем, не находится ли пользователь в режиме добавления заметки
		if state, exists := h.storage.GetUserState(chatID); exists && state == StateWaitingForNoteContent {
			h.notesHandler.HandleNoteContent(chatID, update)
			continue
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

		case "📒 Заметки":
			h.notesHandler.SendNotesMenu(chatID)

		case "📝 Новая заметка":
			h.notesHandler.SendCategoriesForSelection(chatID, "new_note")

		case "📁 Мои заметки":
			// Предлагаем выбрать категорию или показать все заметки
			categories, err := database.GetUserCategories(chatID)
			if err != nil || len(categories) == 0 {
				// Если категорий нет, показываем все заметки
				h.notesHandler.SendUserNotes(chatID, 0)
			} else {
				// Если есть категории, предлагаем выбрать
				h.notesHandler.SendCategoriesForViewing(chatID)
			}

		case "📂 Управление категориями":
			h.notesHandler.SendCategoriesMenu(chatID)

		case "➕ Создать категорию":
			h.notesHandler.AskForCategoryName(chatID)

		case "🗑️ Удалить категорию":
			h.notesHandler.SendCategoriesForSelection(chatID, "delete_category")

		case "✏️ Редактировать категории":
			h.notesHandler.SendEditCategoriesMenu(chatID)

		case "🛠️ Управление заметками":
			h.notesHandler.SendNotesManagementMenu(chatID)

		case "✏️ Редактировать заметку":
			h.notesHandler.SendNotesForSelection(chatID, "edit_note")

		case "🗑️ Удалить заметку":
			h.notesHandler.SendNotesForSelection(chatID, "delete_note")

		case "⬅️ Назад к заметкам":
			h.notesHandler.SendNotesMenu(chatID)

		case "⬅️ Назад к списку":
			h.notesHandler.SendUserNotes(chatID, 0)

		case "⬅️ Назад", "🏠 В начало":
			h.msgHandler.SendMainMenu(chatID)

		case "📸 Медиа-заметки":
			// Получаем ID категории из текущего состояния или используем 0 (все категории)
			userData, exists := h.storage.GetUserData(chatID)
			var categoryID uint = 0
			if exists && userData.Category != "" {
				// Если есть сохраненная категория, используем ее
				if cat, err := database.GetCategoryByName(chatID, userData.Category); err == nil {
					categoryID = cat.ID
				}
			}
			h.notesHandler.SendMediaNotes(chatID, categoryID)

		// В разделе обработки обычных сообщений добавьте:
		case "✏️ Редактировать":
			// Запрашиваем новое содержание
			h.msgHandler.sendMessage(chatID, "📝 Введите новый текст для заметки:", CreateBackKeyboard())
			h.storage.SetUserState(chatID, StateEditingNote)

		case "🗑️ Удалить":
			// Уже обрабатывается в состояниях

		default:
			// Если это медиа-контент или текст (не команда), предлагаем сразу сохранить в заметки
			if update.Message.Photo != nil || update.Message.Video != nil ||
				update.Message.Voice != nil || update.Message.Document != nil ||
				(update.Message.Text != "" && !isCommand(update.Message.Text)) {

				log.Printf("Forwarded message detected: Text=%s, HasPhoto=%t, HasVideo=%t, ForwardFrom=%v",
					update.Message.Text,
					update.Message.Photo != nil,
					update.Message.Video != nil,
					update.Message.ForwardFrom)

				// Сохраняем само сообщение для последующего сохранения
				h.saveMessageForForwarding(chatID, update.Message)

				// Предлагаем выбрать категорию для сохранения
				h.notesHandler.SendCategoriesForSelection(chatID, "save_forwarded_message")
				continue
			}

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

// saveMessageForForwarding сохраняет данные сообщения для последующего сохранения в заметки
func (h *UpdateHandler) saveMessageForForwarding(chatID int64, message *tgbotapi.Message) {
	// Определяем тип контента
	var content string
	var fileID string
	var noteType string

	// Добавляем информацию об источнике
	var sourceInfo string
	if message.ForwardFrom != nil {
		sourceInfo = fmt.Sprintf("От: %s", getUserName(message.ForwardFrom))
	} else if message.ForwardFromChat != nil {
		sourceInfo = fmt.Sprintf("Из: %s", message.ForwardFromChat.Title)
	} else if message.ForwardSenderName != "" {
		sourceInfo = fmt.Sprintf("От: %s", message.ForwardSenderName)
	}

	if message.Text != "" {
		noteType = "text"
		content = message.Text
		if sourceInfo != "" {
			content = sourceInfo + "\n\n" + content
		}
	} else if message.Photo != nil && len(message.Photo) > 0 {
		noteType = "photo"
		fileID = message.Photo[len(message.Photo)-1].FileID
		content = message.Caption
		if sourceInfo != "" {
			if content != "" {
				content = sourceInfo + "\n\n" + content
			} else {
				content = sourceInfo
			}
		}
	} else if message.Video != nil {
		noteType = "video"
		fileID = message.Video.FileID
		content = message.Caption
		if sourceInfo != "" {
			if content != "" {
				content = sourceInfo + "\n\n" + content
			} else {
				content = sourceInfo
			}
		}
	} else if message.Voice != nil {
		noteType = "voice"
		fileID = message.Voice.FileID
		content = sourceInfo // для голосовых сохраняем только источник
	} else if message.Document != nil {
		noteType = "file"
		fileID = message.Document.FileID
		content = message.Caption
		if sourceInfo != "" {
			if content != "" {
				content = sourceInfo + "\n\n" + content
			} else {
				content = sourceInfo
			}
		}
	} else {
		// Если тип не определен, используем текст как fallback
		noteType = "text"
		content = "Пересланное сообщение"
		if sourceInfo != "" {
			content = sourceInfo + "\n\n" + content
		}
	}

	// Сохраняем данные сообщения
	h.storage.SetUserData(chatID, pmodel.UserData{
		Data:     "save_forwarded_message",
		Name:     noteType,
		City:     content,
		Category: fileID,
	})

	log.Printf("Saved forwarded message data: Type=%s, Content=%s, FileID=%s, Source=%s",
		noteType, content, fileID, sourceInfo)
}

// getUserName возвращает имя пользователя для отображения
func getUserName(user *tgbotapi.User) string {
	if user.UserName != "" {
		return "@" + user.UserName
	} else if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}
	return "Пользователь"
}

// isCommand проверяет, является ли текст командой
func isCommand(text string) bool {
	commands := []string{
		"/start", "ℹ️ Информация", "📞 Поддержка", "🌡️Погода", "📒 Заметки",
		"⚙️ Настройки", "🔔 Уведомления", "👤 Профиль", "🌡️ Уведомления о погоде",
		"📝 Новая заметка", "📁 Мои заметки", "📂 Управление категориями",
		"➕ Создать категорию", "🗑️ Удалить категорию", "⬅️ Назад", "🏠 В начало",
		"✏️ Ваше имя", "🚩 Ваш город", "🛠️ Управление заметками",
		"✏️ Редактировать заметку", "🗑️ Удалить заметку", "⬅️ Назад к заметкам",
		"⬅️ Назад к списку", "✏️ Редактировать", "🗑️ Удалить",
		"✏️ Редактировать категории", "➕ Новая категория",
	}

	for _, cmd := range commands {
		if text == cmd {
			return true
		}
	}
	return false
}
