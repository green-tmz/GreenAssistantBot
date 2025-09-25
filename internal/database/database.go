package database

import (
	"GreenAssistantBot/internal/database/models"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	instance *gorm.DB
	once     sync.Once
)

func GetConnect() *gorm.DB {
	once.Do(func() {
		fmt.Println("Connecting to database ...")

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			os.Getenv("DB_USERNAME"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_DATABASE"),
		)

		var err error
		instance, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatal("Database connection error: ", err)
		}

		sqlDB, err := instance.DB()
		if err != nil {
			log.Fatal("Database connection error: ", err)
		}

		err = sqlDB.Ping()
		if err != nil {
			log.Fatal("Database ping error: ", err)
		}

		fmt.Println("Connected to database")
	})

	return instance
}

func AutoMigrate() error {
	db := GetConnect()

	err := db.AutoMigrate(
		&models.User{},
	)

	if err != nil {
		return err
	}

	log.Println("GORM migrations completed successfully")
	return nil
}

func GetUserByTelegramID(telegramID int64) (*models.User, error) {
	db := GetConnect()

	var user models.User
	result := db.Where("telegram_id = ?", telegramID).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

func SaveOrUpdateUser(user *models.User) error {
	db := GetConnect()

	log.Printf("Saving user: TelegramID=%d, FirstName=%s, City=%s",
		user.TelegramID, user.FirstName, user.City)

	// Сначала проверяем, существует ли пользователь
	existingUser, err := GetUserByTelegramID(user.TelegramID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		db.Create(user)
	}

	if existingUser != nil {
		updates := make(map[string]interface{})

		if user.City != "" {
			updates["city"] = user.City
		}
		if user.UserName != "" {
			updates["user_name"] = user.UserName
		}
		if user.FirstName != "" {
			updates["first_name"] = user.FirstName
		}
		if user.LastName != "" {
			updates["last_name"] = user.LastName
		}

		if len(updates) > 0 {
			db.Model(&existingUser).Updates(updates)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func UserExists(telegramID int64) (bool, error) {
	db := GetConnect()

	var count int64
	result := db.Model(&models.User{}).Where("telegram_id = ?", telegramID).Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
