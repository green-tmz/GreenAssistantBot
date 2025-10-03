package models

import (
	"gorm.io/gorm"
	"time"
)

type Category struct {
	gorm.Model
	TelegramID int64  `gorm:"not null"`
	Name       string `gorm:"size:255;not null"`
	Color      string `gorm:"size:50"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Notes []Note `gorm:"foreignKey:CategoryID"`
}
