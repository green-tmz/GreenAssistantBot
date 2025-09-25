package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	TelegramID           int64  `gorm:"uniqueIndex;not null"`
	UserName             string `gorm:"size:255"`
	FirstName            string `gorm:"size:255"`
	LastName             string `gorm:"size:255"`
	City                 string `gorm:"size:255"`
	WeatherNotifications bool   `gorm:"default:true"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
