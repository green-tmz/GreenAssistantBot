package scheduler

import (
	"GreenAssistantBot/internal/bot"
	"GreenAssistantBot/internal/database"
	"GreenAssistantBot/internal/weather"
	"log"
	"os"
	"strconv"
	"time"
)

type Scheduler struct {
	bot            *bot.MessageHandler
	weatherService *weather.WeatherService
}

func NewScheduler(botHandler *bot.MessageHandler) *Scheduler {
	return &Scheduler{
		bot:            botHandler,
		weatherService: weather.NewWeatherService(),
	}
}

// StartWeatherNotifications запускает отправку уведомлений о погоде по расписанию
func (s *Scheduler) StartWeatherNotifications() {
	// Запускаем горутину для проверки времени отправки уведомлений
	go func() {
		for {
			now := time.Now()

			// Получаем время отправки из переменных окружения
			hour := 9 // значение по умолчанию
			if h := os.Getenv("WEATHER_NOTIFICATION_HOUR"); h != "" {
				if parsedHour, err := strconv.Atoi(h); err == nil {
					hour = parsedHour
				}
			}

			minute := 0 // значение по умолчанию
			if m := os.Getenv("WEATHER_NOTIFICATION_MINUTE"); m != "" {
				if parsedMinute, err := strconv.Atoi(m); err == nil {
					minute = parsedMinute
				}
			}

			// Вычисляем время до следующего запланированного времени
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// Ждем до запланированного времени
			duration := nextRun.Sub(now)
			log.Printf("Next weather notification will be sent at %v (in %v)", nextRun, duration)
			time.Sleep(duration)

			// Отправляем уведомления о погоде всем пользователям
			s.sendWeatherToAllUsers()
		}
	}()
}

// sendWeatherToAllUsers отправляет уведомления о погоде всем пользователям
func (s *Scheduler) sendWeatherToAllUsers() {
	// Получаем всех пользователей из базы данных
	users, err := database.GetAllUsers()
	if err != nil {
		log.Printf("Error getting users: %v", err)
		return
	}

	// Отправляем погоду каждому пользователю с включенными уведомлениями
	for _, user := range users {
		if user.City != "" && user.WeatherNotifications {
			weatherData, err := s.weatherService.GetWeatherData(user.City)
			if err != nil {
				log.Printf("Error getting weather data for user %d: %v", user.TelegramID, err)
				continue
			}

			text := "🌅 Доброе утро! Вот прогноз погоды на сегодня:\n\n" + s.weatherService.FormatWeatherMessage(weatherData)
			err = s.bot.SendMessage(user.TelegramID, text, bot.CreateMainMenuKeyboard())
			if err != nil {
				log.Printf("Error sending weather notification to user %d: %v", user.TelegramID, err)
			}
		}
	}
}
