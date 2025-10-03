package models

import (
	"gorm.io/gorm"
	"time"
)

type NoteType string

const (
	NoteTypeText  NoteType = "text"
	NoteTypePhoto NoteType = "photo"
	NoteTypeVideo NoteType = "video"
	NoteTypeVoice NoteType = "voice"
	NoteTypeLink  NoteType = "link"
	NoteTypeFile  NoteType = "file"
)

type Note struct {
	gorm.Model
	TelegramID int64    `gorm:"not null"`
	CategoryID uint     `gorm:"not null"`
	Type       NoteType `gorm:"type:enum('text','photo','video','voice','link','file');not null"`
	Content    string   `gorm:"type:text"`
	FileID     string   `gorm:"size:500"`
	Caption    string   `gorm:"type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Category Category `gorm:"foreignKey:CategoryID"`
}
