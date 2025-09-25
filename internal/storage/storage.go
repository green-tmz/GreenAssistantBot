package storage

import (
	"GreenAssistantBot/pkg/models"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2"
)

// Константы для настройки
const (
	DefaultCacheSize       = 1000
	DefaultCleanupInterval = 5 * time.Minute
)

type BotStorage interface {
	GetUserState(chatID int64) (string, bool)
	SetUserState(chatID int64, state string)
	GetUserData(chatID int64) (models.UserData, bool)
	SetUserData(chatID int64, data models.UserData)
	GetLastMessageID(chatID int64) (int, bool)
	SetLastMessageID(chatID int64, messageID int)
	AddMessageToHistory(chatID int64, messageID int)
	GetMessageHistory(chatID int64) []int
	ClearUserData(chatID int64)
	// Новый метод для периодической очистки
	CleanupExpiredData()
}

type MemoryStorage struct {
	mu sync.RWMutex

	// LRU кэши вместо обычных мап
	userStates      *lru.Cache[int64, string]
	userData        *lru.Cache[int64, models.UserData]
	lastBotMessages *lru.Cache[int64, int]
	messageHistory  *lru.Cache[int64, *messageHistoryEntry]

	// Для TTL (время жизни записей)
	creationTime map[int64]time.Time
}

type messageHistoryEntry struct {
	messageIDs []int
	createdAt  time.Time
}

// NewMemoryStorage создает новое хранилище с ограничением по размеру
func NewMemoryStorage() (*MemoryStorage, error) {
	// Создаем LRU кэши с ограничением размера
	userStates, err := lru.New[int64, string](DefaultCacheSize)
	if err != nil {
		return nil, err
	}

	userData, err := lru.New[int64, models.UserData](DefaultCacheSize)
	if err != nil {
		return nil, err
	}

	lastBotMessages, err := lru.New[int64, int](DefaultCacheSize)
	if err != nil {
		return nil, err
	}

	messageHistory, err := lru.New[int64, *messageHistoryEntry](DefaultCacheSize)
	if err != nil {
		return nil, err
	}

	storage := &MemoryStorage{
		userStates:      userStates,
		userData:        userData,
		lastBotMessages: lastBotMessages,
		messageHistory:  messageHistory,
		creationTime:    make(map[int64]time.Time),
	}

	// Запускаем фоновую очистку
	go storage.startCleanupRoutine()

	return storage, nil
}

// startCleanupRoutine запускает периодическую очистку
func (s *MemoryStorage) startCleanupRoutine() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.CleanupExpiredData()
	}
}

// CleanupExpiredData очищает старые записи
func (s *MemoryStorage) CleanupExpiredData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	maxAge := 24 * time.Hour // Удаляем записи старше 24 часов

	// Очищаем creationTime и удаляем соответствующие данные из кэшей
	for chatID, createdAt := range s.creationTime {
		if now.Sub(createdAt) > maxAge {
			delete(s.creationTime, chatID)
			s.userStates.Remove(chatID)
			s.userData.Remove(chatID)
			s.lastBotMessages.Remove(chatID)
			s.messageHistory.Remove(chatID)
		}
	}
}

// Обновляем все методы с учетом новой структуры:

func (s *MemoryStorage) GetUserState(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.userStates.Get(chatID)
	if exists {
		// Обновляем время доступа для LRU
		s.userStates.Get(chatID) // Это обновляет позицию в LRU
	}
	return state, exists
}

func (s *MemoryStorage) SetUserState(chatID int64, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.userStates.Add(chatID, state)
	s.updateCreationTime(chatID)
}

func (s *MemoryStorage) GetUserData(chatID int64) (models.UserData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.userData.Get(chatID)
	if exists {
		s.userData.Get(chatID) // Обновляем LRU
	}
	return data, exists
}

func (s *MemoryStorage) SetUserData(chatID int64, data models.UserData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.userData.Add(chatID, data)
	s.updateCreationTime(chatID)
}

func (s *MemoryStorage) GetLastMessageID(chatID int64) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messageID, exists := s.lastBotMessages.Get(chatID)
	if exists {
		s.lastBotMessages.Get(chatID)
	}
	return messageID, exists
}

func (s *MemoryStorage) SetLastMessageID(chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastBotMessages.Add(chatID, messageID)
	s.updateCreationTime(chatID)
}

func (s *MemoryStorage) AddMessageToHistory(chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.messageHistory.Get(chatID)
	if !exists {
		entry = &messageHistoryEntry{
			messageIDs: make([]int, 0),
			createdAt:  time.Now(),
		}
	}

	entry.messageIDs = append(entry.messageIDs, messageID)

	// Ограничиваем историю 100 сообщениями
	if len(entry.messageIDs) > 100 {
		entry.messageIDs = entry.messageIDs[len(entry.messageIDs)-100:]
	}

	s.messageHistory.Add(chatID, entry)
	s.updateCreationTime(chatID)
}

func (s *MemoryStorage) GetMessageHistory(chatID int64) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.messageHistory.Get(chatID)
	if !exists {
		return nil
	}

	// Возвращаем копию
	result := make([]int, len(entry.messageIDs))
	copy(result, entry.messageIDs)
	return result
}

func (s *MemoryStorage) ClearUserData(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.userStates.Remove(chatID)
	s.userData.Remove(chatID)
	s.lastBotMessages.Remove(chatID)
	s.messageHistory.Remove(chatID)
	delete(s.creationTime, chatID)
}

// updateCreationTime обновляет время создания/доступа записи
func (s *MemoryStorage) updateCreationTime(chatID int64) {
	if _, exists := s.creationTime[chatID]; !exists {
		s.creationTime[chatID] = time.Now()
	}
}

// GetStats возвращает статистику для мониторинга
func (s *MemoryStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"user_states_size":     s.userStates.Len(),
		"user_data_size":       s.userData.Len(),
		"last_messages_size":   s.lastBotMessages.Len(),
		"message_history_size": s.messageHistory.Len(),
		"active_users":         len(s.creationTime),
		"cache_capacity":       DefaultCacheSize,
	}
}
