package bot

import (
	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/database/models"
	"GreenAssistantBot/internal/storage"
	pmodel "GreenAssistantBot/pkg/models"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type NotesHandler struct {
	bot        *tgbotapi.BotAPI
	storage    storage.BotStorage
	msgHandler *MessageHandler
}

func NewNotesHandler(bot *tgbotapi.BotAPI, storage storage.BotStorage, msgHandler *MessageHandler) *NotesHandler {
	return &NotesHandler{
		bot:        bot,
		storage:    storage,
		msgHandler: msgHandler,
	}
}

// SendNotesMenu отправляет меню заметок
func (h *NotesHandler) SendNotesMenu(chatID int64) {
	text := `📒 **Управление заметками**

Здесь вы можете создавать и организовывать свои заметки по категориям.

✨ **Возможности:**
• 📝 Создание текстовых заметок
• 🖼️ Сохранение фото, видео, голосовых сообщений
• 🔗 Сохранение ссылок и файлов
• 📂 Сортировка по категориям
• 🔍 Быстрый поиск и доступ`

	h.msgHandler.sendMessage(chatID, text, CreateNotesMenuKeyboard())
}

// SendCategoriesMenu отправляет меню категорий
func (h *NotesHandler) SendCategoriesMenu(chatID int64) {
	categories, err := database.GetUserCategories(chatID)
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при загрузке категорий", CreateNotesMenuKeyboard())
		return
	}

	if len(categories) == 0 {
		text := `📂 **Категории**

У вас пока нет категорий. Создайте первую категорию для организации заметок.`
		h.msgHandler.sendMessage(chatID, text, CreateCategoriesManagementKeyboard())
		return
	}

	var categoriesText strings.Builder
	categoriesText.WriteString("📂 **Ваши категории:**\n\n")

	for i, category := range categories {
		count, _ := database.GetNotesCountByCategory(chatID, category.ID)
		emoji := getCategoryEmoji(i)
		categoriesText.WriteString(fmt.Sprintf("%s **%s** - %d заметок\n", emoji, category.Name, count))
	}

	h.msgHandler.sendMessage(chatID, categoriesText.String(), CreateCategoriesManagementKeyboard())
}

// AskForCategoryName запрашивает название новой категории
func (h *NotesHandler) AskForCategoryName(chatID int64) {
	text := "📝 Введите название для новой категории:"
	h.msgHandler.sendMessage(chatID, text, CreateBackKeyboard())
	h.storage.SetUserState(chatID, StateWaitingForCategoryName)
}

// HandleCategoryCreation обрабатывает создание категории
func (h *NotesHandler) HandleCategoryCreation(chatID int64, categoryName string) {
	if strings.TrimSpace(categoryName) == "" {
		h.msgHandler.sendMessage(chatID, "❌ Название категории не может быть пустым", CreateBackKeyboard())
		return
	}

	// Цвета для категорий (можно расширить)
	colors := []string{"🔵", "🟢", "🟡", "🟠", "🔴", "🟣"}
	colorIndex := len(h.storage.GetMessageHistory(chatID)) % len(colors)

	_, err := database.CreateCategory(chatID, categoryName, colors[colorIndex])
	if err != nil {
		log.Printf("Error creating category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при создании категории", CreateNotesMenuKeyboard())
		return
	}

	h.msgHandler.sendMessage(chatID, fmt.Sprintf("✅ Категория \"%s\" успешно создана!", categoryName), CreateCategoriesManagementKeyboard())
	h.storage.SetUserState(chatID, "")
}

// SendCategoriesForSelection отправляет категории для выбора
func (h *NotesHandler) SendCategoriesForSelection(chatID int64, purpose string) {
	categories, err := database.GetUserCategories(chatID)
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		h.msgHandler.SendMessage(chatID, "❌ Ошибка при загрузке категорий", CreateNotesMenuKeyboard())
		return
	}

	if len(categories) == 0 {
		h.msgHandler.SendMessage(chatID, "❌ У вас нет категорий. Сначала создайте категорию.", CreateNotesMenuKeyboard())
		return
	}

	log.Printf("SendCategoriesForSelection: chatID=%d, purpose=%s, categories=%d", chatID, purpose, len(categories))

	// Получаем текущие данные и обновляем только поле Data
	currentData, _ := h.storage.GetUserData(chatID)
	currentData.Data = purpose
	h.storage.SetUserData(chatID, currentData)
	h.storage.SetUserState(chatID, StateWaitingForNoteCategory)

	// Двойная проверка что данные сохранились
	savedData, exists := h.storage.GetUserData(chatID)
	savedState, stateExists := h.storage.GetUserState(chatID)

	log.Printf("After setting - Data exists: %t, State exists: %t", exists, stateExists)
	if exists {
		log.Printf("Saved user data: %+v", savedData)
	}
	if stateExists {
		log.Printf("Saved user state: %s", savedState)
	}

	h.msgHandler.SendMessage(chatID, "📂 Выберите категорию:", CreateCategoriesKeyboard(categories))
}

// HandleNoteContent обрабатывает контент заметки
func (h *NotesHandler) HandleNoteContent(chatID int64, update tgbotapi.Update) {
	userData, exists := h.storage.GetUserData(chatID)
	if !exists {
		log.Printf("No user data found for chat %d in HandleNoteContent", chatID)
		h.msgHandler.sendMessage(chatID, "❌ Сессия истекла, начните заново", CreateNotesMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	categoryName := userData.Category
	log.Printf("Processing note content for category: %s", categoryName)

	// Находим категорию по имени используя новую функцию
	selectedCategory, err := database.GetCategoryByName(chatID, categoryName)
	if err != nil {
		log.Printf("Error finding category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateNotesMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	log.Printf("Selected category: %+v", selectedCategory)

	// Создаем заметку в зависимости от типа контента
	note := &models.Note{
		TelegramID: chatID,
		CategoryID: selectedCategory.ID,
	}

	message := update.Message
	if message == nil {
		log.Printf("Message is nil")
		return
	}

	// Определяем тип контента и сохраняем
	if message.Text != "" {
		note.Type = models.NoteTypeText
		note.Content = message.Text
		log.Printf("Creating text note: %s", message.Text)
	} else if message.Photo != nil && len(message.Photo) > 0 {
		note.Type = models.NoteTypePhoto
		note.FileID = message.Photo[len(message.Photo)-1].FileID
		note.Caption = message.Caption
		log.Printf("Creating photo note with file ID: %s", note.FileID)
	} else if message.Video != nil {
		note.Type = models.NoteTypeVideo
		note.FileID = message.Video.FileID
		note.Caption = message.Caption
		log.Printf("Creating video note with file ID: %s", note.FileID)
	} else if message.Voice != nil {
		note.Type = models.NoteTypeVoice
		note.FileID = message.Voice.FileID
		log.Printf("Creating voice note with file ID: %s", note.FileID)
	} else if message.Document != nil {
		note.Type = models.NoteTypeFile
		note.FileID = message.Document.FileID
		note.Caption = message.Caption
		log.Printf("Creating file note with file ID: %s", note.FileID)
	} else {
		log.Printf("Unsupported message type")
		h.msgHandler.sendMessage(chatID, "❌ Неподдерживаемый тип сообщения", CreateNotesMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	if err := database.CreateNote(note); err != nil {
		log.Printf("Error creating note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при сохранении заметки", CreateNotesMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	log.Printf("Note created successfully")
	h.msgHandler.sendMessage(chatID,
		fmt.Sprintf("✅ Заметка сохранена в категорию \"%s\"!", selectedCategory.Name),
		CreateNotesMenuKeyboard())
	h.storage.SetUserState(chatID, "")
}

// SendNotesByCategory отправляет заметки конкретной категории
func (h *NotesHandler) SendNotesByCategory(chatID int64, categoryName string) {
	category, err := database.GetCategoryByName(chatID, categoryName)
	if err != nil {
		log.Printf("Error finding category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateNotesMenuKeyboard())
		return
	}

	h.SendUserNotes(chatID, category.ID)
}

// SendCategoriesForViewing отправляет категории для просмотра заметок
func (h *NotesHandler) SendCategoriesForViewing(chatID int64) {
	h.SendCategoriesForSelection(chatID, "view_notes")
}

// SendMediaNotes отправляет только медиа-заметки
func (h *NotesHandler) SendMediaNotes(chatID int64, categoryID uint) {
	notes, err := database.GetUserNotes(chatID, categoryID)
	if err != nil {
		log.Printf("Error getting notes: %v", err)
		return
	}

	// Фильтруем только медиа-заметки
	var mediaNotes []models.Note
	for _, note := range notes {
		if note.Type == models.NoteTypePhoto || note.Type == models.NoteTypeVideo || note.Type == models.NoteTypeVoice {
			mediaNotes = append(mediaNotes, note)
		}
	}

	if len(mediaNotes) == 0 {
		h.msgHandler.sendMessage(chatID, "📸 В этой категории нет медиа-заметок", CreateNotesViewKeyboard())
		return
	}

	// Отправляем информацию о количестве медиа-заметок
	countMsg := fmt.Sprintf("📸 Медиа-заметок: %d", len(mediaNotes))
	h.msgHandler.sendMessage(chatID, countMsg, CreateNotesViewKeyboard())

	// Отправляем все медиа-заметки
	for _, note := range mediaNotes {
		h.sendNotePreview(chatID, note)
		// Задержка между отправкой медиа
		time.Sleep(500 * time.Millisecond)
	}
}

// SendUserNotes отправляет заметки пользователя
func (h *NotesHandler) SendUserNotes(chatID int64, categoryID uint) {
	notes, err := database.GetUserNotes(chatID, categoryID)
	if err != nil {
		log.Printf("Error getting notes: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при загрузке заметок", CreateNotesMenuKeyboard())
		return
	}

	if len(notes) == 0 {
		var msg string
		if categoryID == 0 {
			msg = "📝 У вас пока нет заметок"
		} else {
			category, _ := database.GetCategoryByID(chatID, categoryID)
			if category != nil {
				msg = fmt.Sprintf("📝 У вас пока нет заметок в категории \"%s\"", category.Name)
			} else {
				msg = "📝 У вас пока нет заметок в этой категории"
			}
		}
		h.msgHandler.sendMessage(chatID, msg, CreateNotesMenuKeyboard())
		return
	}

	// Отправляем информацию о количестве заметок
	var countMsg string
	if categoryID == 0 {
		countMsg = fmt.Sprintf("📋 Всего заметок: %d", len(notes))
	} else {
		category, _ := database.GetCategoryByID(chatID, categoryID)
		if category != nil {
			countMsg = fmt.Sprintf("📋 Заметок в категории \"%s\": %d", category.Name, len(notes))
		} else {
			countMsg = fmt.Sprintf("📋 Заметок: %d", len(notes))
		}
	}
	h.msgHandler.sendMessage(chatID, countMsg, CreateNotesViewKeyboard())

	// Отправляем все заметки по порядку
	for _, note := range notes {
		h.sendNotePreview(chatID, note)
		// Небольшая задержка между отправкой заметок
		time.Sleep(300 * time.Millisecond)
	}

	// Добавляем кнопку управления заметками
	h.msgHandler.sendMessage(chatID, "🛠️ Для управления заметками используйте меню управления", CreateNotesViewKeyboard())
}

// sendNotePreview отправляет превью заметки
func (h *NotesHandler) sendNotePreview(chatID int64, note models.Note) {
	var text string
	emoji := getNoteTypeEmoji(note.Type)

	switch note.Type {
	case models.NoteTypeText:
		// Отправляем полный текст без обрезания
		text = fmt.Sprintf("%s **Текстовая заметка**\n📂 Категория: %s\n📅 %s\n\n%s",
			emoji, note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"), note.Content)
		h.sendLongMessage(chatID, text)

	case models.NoteTypePhoto:
		text = fmt.Sprintf("%s **Фото заметка**\n📂 Категория: %s\n📅 %s",
			emoji, note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))
		if note.Caption != "" {
			text += fmt.Sprintf("\n📝 Подпись: %s", note.Caption)
		}
		// Отправляем фото
		h.sendMediaMessage(chatID, note.FileID, "photo", text)

	case models.NoteTypeVideo:
		text = fmt.Sprintf("%s **Видео заметка**\n📂 Категория: %s\n📅 %s",
			emoji, note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))
		if note.Caption != "" {
			text += fmt.Sprintf("\n📝 Подпись: %s", note.Caption)
		}
		// Отправляем видео
		h.sendMediaMessage(chatID, note.FileID, "video", text)

	case models.NoteTypeVoice:
		text = fmt.Sprintf("%s **Голосовая заметка**\n📂 Категория: %s\n📅 %s",
			emoji, note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))
		// Отправляем голосовое сообщение
		h.sendMediaMessage(chatID, note.FileID, "voice", text)

	case models.NoteTypeFile:
		text = fmt.Sprintf("%s **Файл**\n📂 Категория: %s\n📅 %s",
			emoji, note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))
		if note.Caption != "" {
			text += fmt.Sprintf("\n📝 Подпись: %s", note.Caption)
		}
		h.sendLongMessage(chatID, text)

	default:
		text = fmt.Sprintf("%s **%s заметка**\n📂 Категория: %s\n📅 %s",
			emoji, strings.Title(string(note.Type)), note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))
		if note.Caption != "" {
			text += fmt.Sprintf("\n📝 Подпись: %s", note.Caption)
		}
		h.sendLongMessage(chatID, text)
	}
}

// sendLongMessage отправляет длинное сообщение, разбивая его на части если нужно
func (h *NotesHandler) sendLongMessage(chatID int64, text string) {
	// Максимальная длина сообщения в Telegram
	maxLength := 4096

	if len(text) <= maxLength {
		h.msgHandler.sendMessage(chatID, text, CreateNotesMenuKeyboard())
		return
	}

	// Разбиваем текст на части
	parts := splitMessage(text, maxLength)

	// Отправляем первую часть с клавиатурой
	if len(parts) > 0 {
		h.msgHandler.sendMessage(chatID, parts[0], CreateNotesMenuKeyboard())
	}

	// Отправляем остальные части без клавиатуры
	for i := 1; i < len(parts); i++ {
		h.msgHandler.sendMessage(chatID, parts[i], tgbotapi.NewRemoveKeyboard(true))
		// Небольшая задержка между сообщениями
		if i < len(parts)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// splitMessage разбивает текст на части указанной максимальной длины
func splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= maxLength {
			parts = append(parts, text)
			break
		}

		// Пытаемся разбить по переносу строки
		splitIndex := strings.LastIndex(text[:maxLength], "\n")
		if splitIndex == -1 {
			// Если переносов строки нет, разбиваем по пробелу
			splitIndex = strings.LastIndex(text[:maxLength], " ")
			if splitIndex == -1 {
				// Если пробелов нет, просто обрезаем
				splitIndex = maxLength
			}
		}

		parts = append(parts, text[:splitIndex])
		text = text[splitIndex:]

		// Убираем начальные пробелы/переносы
		text = strings.TrimLeft(text, " \n")
	}

	return parts
}

// sendMediaMessage отправляет медиа-файл с подписью
func (h *NotesHandler) sendMediaMessage(chatID int64, fileID, mediaType, caption string) {
	// Убираем ограничение длины подписи
	// Telegram сам обрежет слишком длинные подписи

	var err error
	switch mediaType {
	case "photo":
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(fileID))
		photo.Caption = caption
		photo.ParseMode = "Markdown"
		_, err = h.bot.Send(photo)

	case "video":
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileID(fileID))
		video.Caption = caption
		video.ParseMode = "Markdown"
		_, err = h.bot.Send(video)

	case "voice":
		voice := tgbotapi.NewVoice(chatID, tgbotapi.FileID(fileID))
		voice.Caption = caption
		voice.ParseMode = "Markdown"
		_, err = h.bot.Send(voice)

	default:
		log.Printf("Unsupported media type: %s", mediaType)
		return
	}

	if err != nil {
		log.Printf("Error sending media message: %v", err)
		// Если не удалось отправить медиа, отправляем текстовое описание
		h.sendLongMessage(chatID, caption)
	}
}

// HandleDeleteCategory обрабатывает удаление категории
func (h *NotesHandler) HandleDeleteCategory(chatID int64, categoryName string) {
	categories, err := database.GetUserCategories(chatID)
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при загрузке категорий", CreateNotesMenuKeyboard())
		return
	}

	var categoryToDelete *models.Category
	for _, cat := range categories {
		if cat.Name == categoryName {
			categoryToDelete = &cat
			break
		}
	}

	if categoryToDelete == nil {
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateNotesMenuKeyboard())
		return
	}

	// Подтверждение удаления
	notesCount, _ := database.GetNotesCountByCategory(chatID, categoryToDelete.ID)
	text := fmt.Sprintf("⚠️ **Подтверждение удаления**\n\nКатегория: **%s**\nКоличество заметок: **%d**\n\nВсе заметки в этой категории будут удалены безвозвратно.\n\nПожалуйста, подтвердите удаление.",
		categoryToDelete.Name, notesCount)

	// Используем клавиатуру подтверждения вместо обычной клавиатуры "Назад"
	h.msgHandler.sendMessage(chatID, text, CreateConfirmationKeyboard())
	h.storage.SetUserData(chatID, pmodel.UserData{Data: strconv.FormatUint(uint64(categoryToDelete.ID), 10)})
	h.storage.SetUserState(chatID, StateDeletingCategory)
}

// ConfirmDeleteCategory подтверждает удаление категории
func (h *NotesHandler) ConfirmDeleteCategory(chatID int64, confirm bool) {
	if !confirm {
		h.msgHandler.sendMessage(chatID, "❌ Удаление отменено", CreateCategoriesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	userData, _ := h.storage.GetUserData(chatID)
	categoryID, err := strconv.ParseUint(userData.Data, 10, 32)
	if err != nil {
		log.Printf("Error parsing category ID: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при удалении категории", CreateCategoriesManagementKeyboard())
		return
	}

	if err := database.DeleteCategory(chatID, uint(categoryID)); err != nil {
		log.Printf("Error deleting category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при удалении категории", CreateCategoriesManagementKeyboard())
		return
	}

	h.msgHandler.sendMessage(chatID, "✅ Категория и все связанные заметки удалены", CreateCategoriesManagementKeyboard())
	h.storage.SetUserState(chatID, "")
}

// Вспомогательные функции
func getCategoryEmoji(index int) string {
	emojis := []string{"🔵", "🟢", "🟡", "🟠", "🔴", "🟣", "⚫️", "⚪️"}
	return emojis[index%len(emojis)]
}

func getNoteTypeEmoji(noteType models.NoteType) string {
	switch noteType {
	case models.NoteTypeText:
		return "📝"
	case models.NoteTypePhoto:
		return "🖼️"
	case models.NoteTypeVideo:
		return "🎥"
	case models.NoteTypeVoice:
		return "🎤"
	case models.NoteTypeLink:
		return "🔗"
	case models.NoteTypeFile:
		return "📎"
	default:
		return "📄"
	}
}

// SendEditCategoriesMenu отправляет меню редактирования категорий
func (h *NotesHandler) SendEditCategoriesMenu(chatID int64) {
	categories, err := database.GetUserCategories(chatID)
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		h.msgHandler.SendMessage(chatID, "❌ Ошибка при загрузке категорий", CreateCategoriesManagementKeyboard())
		return
	}

	if len(categories) == 0 {
		text := `📂 **Редактирование категорий**

У вас пока нет категорий для редактирования. Создайте первую категорию.`
		h.msgHandler.SendMessage(chatID, text, CreateCategoriesManagementKeyboard())
		return
	}

	var categoriesText strings.Builder
	categoriesText.WriteString("📂 **Редактирование категорий**\n\n")
	categoriesText.WriteString("Выберите категорию для редактирования:\n\n")

	for i, category := range categories {
		count, _ := database.GetNotesCountByCategory(chatID, category.ID)
		emoji := getCategoryEmoji(i)
		categoriesText.WriteString(fmt.Sprintf("%s **%s** - %d заметок\n", emoji, category.Name, count))
	}

	// Сохраняем цель выбора категории
	h.storage.SetUserData(chatID, pmodel.UserData{Data: "edit_category"})
	h.storage.SetUserState(chatID, StateWaitingForNoteCategory)

	// Отправляем сообщение ПОСЛЕ установки состояния и данных
	h.msgHandler.SendMessage(chatID, categoriesText.String(), CreateCategoriesKeyboard(categories))
}

// HandleEditCategory обрабатывает редактирование категории
func (h *NotesHandler) HandleEditCategory(chatID int64, categoryName string) {
	category, err := database.GetCategoryByName(chatID, categoryName)
	if err != nil {
		log.Printf("Error finding category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateCategoriesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Сохраняем ID категории для редактирования
	h.storage.SetUserData(chatID, pmodel.UserData{
		Data: strconv.FormatUint(uint64(category.ID), 10),
	})

	text := fmt.Sprintf("✏️ **Редактирование категории**\n\n📂 Текущее название: **%s**\n🎨 Цвет: %s\n\nВведите новое название для категории:",
		category.Name, category.Color)

	h.msgHandler.sendMessage(chatID, text, CreateBackKeyboard())
	h.storage.SetUserState(chatID, StateEditingCategory)
}

// HandleCategoryUpdate обрабатывает обновление названия категории
func (h *NotesHandler) HandleCategoryUpdate(chatID int64, newName string) {
	if strings.TrimSpace(newName) == "" {
		h.msgHandler.sendMessage(chatID, "❌ Название категории не может быть пустым", CreateBackKeyboard())
		return
	}

	userData, _ := h.storage.GetUserData(chatID)
	categoryID, err := strconv.ParseUint(userData.Data, 10, 32)
	if err != nil {
		log.Printf("Error parsing category ID: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при обновлении категории", CreateCategoriesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Получаем текущую категорию
	category, err := database.GetCategoryByID(chatID, uint(categoryID))
	if err != nil {
		log.Printf("Error getting category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateCategoriesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Обновляем название
	category.Name = newName
	db := database.GetConnect()
	if err := db.Save(category).Error; err != nil {
		log.Printf("Error updating category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при обновлении категории", CreateCategoriesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	h.msgHandler.sendMessage(chatID, fmt.Sprintf("✅ Категория успешно переименована в \"%s\"", newName), CreateCategoriesManagementKeyboard())
	h.storage.SetUserState(chatID, "")
}

// SendNotesManagementMenu отправляет меню управления заметками
func (h *NotesHandler) SendNotesManagementMenu(chatID int64) {
	text := `🛠️ **Управление заметками**

Здесь вы можете редактировать и удалять существующие заметки.

✨ **Доступные действия:**
• ✏️ Редактировать заметку - изменить содержание или категорию
• 🗑️ Удалить заметку - безвозвратно удалить заметку`

	h.msgHandler.sendMessage(chatID, text, CreateNotesManagementKeyboard())
}

// SendNotesForSelection отправляет список заметок для выбора
func (h *NotesHandler) SendNotesForSelection(chatID int64, purpose string) {
	notes, err := database.GetUserNotes(chatID, 0) // 0 - все категории
	if err != nil {
		log.Printf("Error getting notes: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при загрузке заметок", CreateNotesManagementKeyboard())
		return
	}

	if len(notes) == 0 {
		h.msgHandler.sendMessage(chatID, "❌ У вас пока нет заметок", CreateNotesManagementKeyboard())
		return
	}

	// Сохраняем цель выбора заметки
	h.storage.SetUserData(chatID, pmodel.UserData{Data: purpose})
	h.storage.SetUserState(chatID, StateWaitingForNoteSelection)

	// Отправляем список заметок
	var notesText strings.Builder
	notesText.WriteString("📋 **Выберите заметку:**\n\n")

	for i, note := range notes {
		if i >= 10 { // Ограничиваем показ 10 заметками
			notesText.WriteString(fmt.Sprintf("\n... и еще %d заметок", len(notes)-10))
			break
		}

		var preview string
		switch note.Type {
		case models.NoteTypeText:
			if len(note.Content) > 50 {
				preview = note.Content[:50] + "..."
			} else {
				preview = note.Content
			}
		case models.NoteTypePhoto:
			preview = "🖼️ Фото"
		case models.NoteTypeVideo:
			preview = "🎥 Видео"
		case models.NoteTypeVoice:
			preview = "🎤 Голосовое сообщение"
		case models.NoteTypeFile:
			preview = "📎 Файл"
		default:
			preview = "📄 Заметка"
		}

		emoji := getNoteTypeEmoji(note.Type)
		notesText.WriteString(fmt.Sprintf("%s `%d`: %s\n", emoji, note.ID, preview))
	}

	h.msgHandler.sendMessage(chatID, notesText.String(), CreateBackKeyboard())
}

// HandleEditNoteSelection обрабатывает выбор заметки для редактирования
func (h *NotesHandler) HandleEditNoteSelection(chatID int64, noteID uint) {
	note, err := database.GetNoteByID(chatID, noteID)
	if err != nil {
		log.Printf("Error getting note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Заметка не найдена", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Сохраняем ID заметки для редактирования
	h.storage.SetUserData(chatID, pmodel.UserData{
		Data: strconv.FormatUint(uint64(noteID), 10),
	})

	// Показываем информацию о заметке и действия
	text := fmt.Sprintf("✏️ **Редактирование заметки**\n\n%s\n\n📂 Категория: %s\n📅 Создана: %s",
		h.formatNoteContent(note), note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))

	h.msgHandler.sendMessage(chatID, text, CreateNoteActionsKeyboard())
	h.storage.SetUserState(chatID, "")
}

// HandleDeleteNoteSelection обрабатывает выбор заметки для удаления
func (h *NotesHandler) HandleDeleteNoteSelection(chatID int64, noteID uint) {
	note, err := database.GetNoteByID(chatID, noteID)
	if err != nil {
		log.Printf("Error getting note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Заметка не найдена", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Сохраняем ID заметки для удаления
	h.storage.SetUserData(chatID, pmodel.UserData{
		Data: strconv.FormatUint(uint64(noteID), 10),
	})

	// Подтверждение удаления
	text := fmt.Sprintf("⚠️ **Подтверждение удаления**\n\n%s\n\n📂 Категория: %s\n📅 Создана: %s\n\nЗаметка будет удалена безвозвратно.\n\nПожалуйста, подтвердите удаление.",
		h.formatNoteContent(note), note.Category.Name, note.CreatedAt.Format("02.01.2006 15:04"))

	h.msgHandler.sendMessage(chatID, text, CreateConfirmationKeyboard())
	h.storage.SetUserState(chatID, StateDeletingNote)
}

// ConfirmDeleteNote подтверждает удаление заметки
func (h *NotesHandler) ConfirmDeleteNote(chatID int64, confirm bool) {
	if !confirm {
		h.msgHandler.sendMessage(chatID, "❌ Удаление отменено", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	userData, _ := h.storage.GetUserData(chatID)
	noteID, err := strconv.ParseUint(userData.Data, 10, 32)
	if err != nil {
		log.Printf("Error parsing note ID: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при удалении заметки", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	if err := database.DeleteNote(chatID, uint(noteID)); err != nil {
		log.Printf("Error deleting note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при удалении заметки", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	h.msgHandler.sendMessage(chatID, "✅ Заметка успешно удалена", CreateNotesManagementKeyboard())
	h.storage.SetUserState(chatID, "")
}

// HandleNoteContentUpdate обрабатывает обновление содержания заметки
func (h *NotesHandler) HandleNoteContentUpdate(chatID int64, newContent string) {
	userData, _ := h.storage.GetUserData(chatID)
	noteID, err := strconv.ParseUint(userData.Data, 10, 32)
	if err != nil {
		log.Printf("Error parsing note ID: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при обновлении заметки", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	note, err := database.GetNoteByID(chatID, uint(noteID))
	if err != nil {
		log.Printf("Error getting note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Заметка не найдена", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Обновляем содержание
	note.Content = newContent
	db := database.GetConnect()
	if err := db.Save(note).Error; err != nil {
		log.Printf("Error updating note: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при обновлении заметки", CreateNotesManagementKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	h.msgHandler.sendMessage(chatID, "✅ Заметка успешно обновлена", CreateNotesManagementKeyboard())
	h.storage.SetUserState(chatID, "")
}

// formatNoteContent форматирует содержание заметки для отображения
func (h *NotesHandler) formatNoteContent(note *models.Note) string {
	switch note.Type {
	case models.NoteTypeText:
		if len(note.Content) > 100 {
			return note.Content[:100] + "..."
		}
		return note.Content
	case models.NoteTypePhoto:
		if note.Caption != "" {
			return fmt.Sprintf("🖼️ Фото: %s", note.Caption)
		}
		return "🖼️ Фото"
	case models.NoteTypeVideo:
		if note.Caption != "" {
			return fmt.Sprintf("🎥 Видео: %s", note.Caption)
		}
		return "🎥 Видео"
	case models.NoteTypeVoice:
		return "🎤 Голосовое сообщение"
	case models.NoteTypeFile:
		if note.Caption != "" {
			return fmt.Sprintf("📎 Файл: %s", note.Caption)
		}
		return "📎 Файл"
	default:
		return "📄 Заметка"
	}
}

// SaveForwardedMessage сохраняет пересланное сообщение в выбранной категории
func (h *NotesHandler) SaveForwardedMessage(chatID int64, categoryName string, userData pmodel.UserData) {
	// Находим категорию
	category, err := database.GetCategoryByName(chatID, categoryName)
	if err != nil {
		log.Printf("Error finding category: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Категория не найдена", CreateMainMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Преобразуем строковый тип в models.NoteType
	var noteType models.NoteType
	switch userData.Name {
	case "text":
		noteType = models.NoteTypeText
	case "photo":
		noteType = models.NoteTypePhoto
	case "video":
		noteType = models.NoteTypeVideo
	case "voice":
		noteType = models.NoteTypeVoice
	case "file":
		noteType = models.NoteTypeFile
	default:
		noteType = models.NoteTypeText // значение по умолчанию
	}

	// Создаем заметку на основе сохраненных данных
	note := &models.Note{
		TelegramID: chatID,
		CategoryID: category.ID,
		Type:       noteType,
		FileID:     userData.Category, // FileID для медиа
	}

	// Для текстовых заметок используем Content, для медиа - Caption
	if noteType == models.NoteTypeText {
		note.Content = userData.City
	} else {
		note.Caption = userData.City
		// Для медиа также сохраняем текст в Content для поиска
		note.Content = userData.City
	}

	log.Printf("Creating note: Type=%s, Content=%s, FileID=%s, CategoryID=%d",
		note.Type, note.Content, note.FileID, note.CategoryID)

	if err := database.CreateNote(note); err != nil {
		log.Printf("Error creating note from forwarded message: %v", err)
		h.msgHandler.sendMessage(chatID, "❌ Ошибка при сохранении заметки: "+err.Error(), CreateMainMenuKeyboard())
		h.storage.SetUserState(chatID, "")
		return
	}

	// Формируем сообщение об успехе
	successMsg := h.createSuccessMessage(note, category.Name)
	h.msgHandler.sendMessage(chatID, successMsg, CreateMainMenuKeyboard())
	h.storage.SetUserState(chatID, "")
}

func (h *NotesHandler) createSuccessMessage(note *models.Note, categoryName string) string {
	var successMsg string

	switch note.Type {
	case models.NoteTypeText:
		successMsg = fmt.Sprintf("✅ Текст сохранен в категорию \"%s\"!\n\n%s", categoryName, note.Content)

	case models.NoteTypePhoto:
		successMsg = fmt.Sprintf("✅ Фото сохранено в категорию \"%s\"!", categoryName)
		if note.Caption != "" {
			successMsg += "\n\n📝 " + note.Caption
		}
		// Отправляем само фото для подтверждения
		go h.sendMediaPreview(note.TelegramID, note.FileID, "photo")

	case models.NoteTypeVideo:
		successMsg = fmt.Sprintf("✅ Видео сохранено в категорию \"%s\"!", categoryName)
		if note.Caption != "" {
			successMsg += "\n\n📝 " + note.Caption
		}

	case models.NoteTypeVoice:
		successMsg = fmt.Sprintf("✅ Голосовое сообщение сохранено в категорию \"%s\"!", categoryName)
		if note.Caption != "" {
			successMsg += "\n\n📝 " + note.Caption
		}

	case models.NoteTypeFile:
		successMsg = fmt.Sprintf("✅ Файл сохранен в категорию \"%s\"!", categoryName)
		if note.Caption != "" {
			successMsg += "\n\n📝 " + note.Caption
		}

	default:
		successMsg = fmt.Sprintf("✅ Сообщение сохранено в категорию \"%s\"!", categoryName)
	}

	return successMsg
}

// sendMediaPreview отправляет превью медиа-файла
func (h *NotesHandler) sendMediaPreview(chatID int64, fileID string, mediaType string) {
	switch mediaType {
	case "photo":
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(fileID))
		photo.Caption = "📸 Сохраненное фото"
		_, err := h.bot.Send(photo)
		if err != nil {
			log.Printf("Error sending photo preview: %v", err)
		}
	case "video":
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileID(fileID))
		video.Caption = "🎥 Сохраненное видео"
		_, err := h.bot.Send(video)
		if err != nil {
			log.Printf("Error sending video preview: %v", err)
		}
	}
}
