package storage

import (
	"GreenAssistantBot/pkg/models"
	"log"
	"sync"
	"time"
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
	CleanupExpiredData()
}

type MemoryStorage struct {
	mu sync.RWMutex

	// Простые мапы вместо LRU для надежности
	userStates      map[int64]string
	userData        map[int64]models.UserData
	lastBotMessages map[int64]int
	messageHistory  map[int64][]int

	// Время последнего доступа для очистки
	lastAccess map[int64]time.Time
}

func NewMemoryStorage() (*MemoryStorage, error) {
	storage := &MemoryStorage{
		userStates:      make(map[int64]string),
		userData:        make(map[int64]models.UserData),
		lastBotMessages: make(map[int64]int),
		messageHistory:  make(map[int64][]int),
		lastAccess:      make(map[int64]time.Time),
	}

	// Запускаем фоновую очистку
	go storage.startCleanupRoutine()

	return storage, nil
}

func (s *MemoryStorage) startCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.CleanupExpiredData()
	}
}

func (s *MemoryStorage) CleanupExpiredData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	maxAge := 24 * time.Hour // Удаляем записи старше 24 часов

	for chatID, lastAccess := range s.lastAccess {
		if now.Sub(lastAccess) > maxAge {
			delete(s.userStates, chatID)
			delete(s.userData, chatID)
			delete(s.lastBotMessages, chatID)
			delete(s.messageHistory, chatID)
			delete(s.lastAccess, chatID)
		}
	}
}

func (s *MemoryStorage) updateLastAccess(chatID int64) {
	s.lastAccess[chatID] = time.Now()
}

func (s *MemoryStorage) GetUserState(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.userStates[chatID]
	if exists {
		s.updateLastAccess(chatID)
		log.Printf("Storage: GetUserState for chat %d: %s", chatID, state)
	} else {
		log.Printf("Storage: GetUserState for chat %d: NOT FOUND", chatID)
	}
	return state, exists
}

func (s *MemoryStorage) SetUserState(chatID int64, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Storage: SetUserState for chat %d: %s", chatID, state)
	s.userStates[chatID] = state
	s.updateLastAccess(chatID)
}

func (s *MemoryStorage) GetUserData(chatID int64) (models.UserData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.userData[chatID]
	if exists {
		s.updateLastAccess(chatID)
		log.Printf("Storage: GetUserData for chat %d: Name='%s', City='%s', Data='%s', Category='%s'",
			chatID, data.Name, data.City, data.Data, data.Category)
	} else {
		log.Printf("Storage: GetUserData for chat %d: NOT FOUND", chatID)
	}
	return data, exists
}

func (s *MemoryStorage) SetUserData(chatID int64, data models.UserData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Storage: SetUserData for chat %d: Name='%s', City='%s', Data='%s', Category='%s'",
		chatID, data.Name, data.City, data.Data, data.Category)
	s.userData[chatID] = data
	s.updateLastAccess(chatID)
	log.Printf("Storage: User data set successfully for chat %d", chatID)
}

func (s *MemoryStorage) GetLastMessageID(chatID int64) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messageID, exists := s.lastBotMessages[chatID]
	if exists {
		s.updateLastAccess(chatID)
	}
	return messageID, exists
}

func (s *MemoryStorage) SetLastMessageID(chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastBotMessages[chatID] = messageID
	s.updateLastAccess(chatID)
}

func (s *MemoryStorage) AddMessageToHistory(chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.messageHistory[chatID] == nil {
		s.messageHistory[chatID] = make([]int, 0)
	}

	s.messageHistory[chatID] = append(s.messageHistory[chatID], messageID)

	// Ограничиваем историю 100 сообщениями
	if len(s.messageHistory[chatID]) > 100 {
		s.messageHistory[chatID] = s.messageHistory[chatID][len(s.messageHistory[chatID])-100:]
	}

	s.updateLastAccess(chatID)
}

func (s *MemoryStorage) GetMessageHistory(chatID int64) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.messageHistory[chatID]
	if history == nil {
		return nil
	}

	// Возвращаем копию
	result := make([]int, len(history))
	copy(result, history)
	return result
}

func (s *MemoryStorage) ClearUserData(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.userStates, chatID)
	delete(s.userData, chatID)
	delete(s.lastBotMessages, chatID)
	delete(s.messageHistory, chatID)
	delete(s.lastAccess, chatID)
	log.Printf("Storage: Cleared all data for chat %d", chatID)
}

func (s *MemoryStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"user_states_size":     len(s.userStates),
		"user_data_size":       len(s.userData),
		"last_messages_size":   len(s.lastBotMessages),
		"message_history_size": len(s.messageHistory),
		"active_users":         len(s.lastAccess),
	}
}
