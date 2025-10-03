package database

import (
	"GreenAssistantBot/internal/database/models"
	"errors"

	"gorm.io/gorm"
)

// Category operations
func CreateCategory(telegramID int64, name, color string) (*models.Category, error) {
	db := GetConnect()

	category := &models.Category{
		TelegramID: telegramID,
		Name:       name,
		Color:      color,
	}

	result := db.Create(category)
	if result.Error != nil {
		return nil, result.Error
	}

	return category, nil
}

func GetUserCategories(telegramID int64) ([]models.Category, error) {
	db := GetConnect()

	var categories []models.Category
	result := db.Where("telegram_id = ? AND deleted_at IS NULL", telegramID).Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}

	return categories, nil
}

func GetCategoryByID(telegramID int64, categoryID uint) (*models.Category, error) {
	db := GetConnect()

	var category models.Category
	result := db.Where("telegram_id = ? AND id = ?", telegramID, categoryID).First(&category)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &category, nil
}

func DeleteCategory(telegramID int64, categoryID uint) error {
	db := GetConnect()

	// Удаляем все заметки в категории
	if err := db.Where("telegram_id = ? AND category_id = ?", telegramID, categoryID).Delete(&models.Note{}).Error; err != nil {
		return err
	}

	// Удаляем саму категорию
	result := db.Where("telegram_id = ? AND id = ?", telegramID, categoryID).Delete(&models.Category{})
	return result.Error
}

// Note operations
func CreateNote(note *models.Note) error {
	db := GetConnect()
	result := db.Create(note)
	return result.Error
}

func GetUserNotes(telegramID int64, categoryID uint) ([]models.Note, error) {
	db := GetConnect()

	var notes []models.Note
	query := db.Where("telegram_id = ?", telegramID)

	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	// Исключаем удаленные заметки (deleted_at IS NULL)
	result := query.Where("deleted_at IS NULL").Preload("Category").Order("created_at ASC").Find(&notes)
	if result.Error != nil {
		return nil, result.Error
	}

	return notes, nil
}

func GetNoteByID(telegramID int64, noteID uint) (*models.Note, error) {
	db := GetConnect()

	var note models.Note
	result := db.Where("telegram_id = ? AND id = ? AND deleted_at IS NULL", telegramID, noteID).Preload("Category").First(&note)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &note, nil
}

func DeleteNote(telegramID int64, noteID uint) error {
	db := GetConnect()
	result := db.Where("telegram_id = ? AND id = ?", telegramID, noteID).Delete(&models.Note{})
	return result.Error
}

func GetNotesCountByCategory(telegramID int64, categoryID uint) (int64, error) {
	db := GetConnect()
	var count int64
	result := db.Model(&models.Note{}).Where("telegram_id = ? AND category_id = ? AND deleted_at IS NULL",
		telegramID, categoryID).Count(&count)
	return count, result.Error
}

// GetCategoryByName ищет категорию по имени для конкретного пользователя
func GetCategoryByName(telegramID int64, categoryName string) (*models.Category, error) {
	db := GetConnect()

	var category models.Category
	result := db.Where("telegram_id = ? AND name = ? AND deleted_at IS NULL", telegramID, categoryName).First(&category)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &category, nil
}
