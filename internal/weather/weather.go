package weather

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// WeatherData — структура под JSON-ответ OpenWeatherMap
type WeatherData struct {
	Name string `json:"name"`
	Main struct {
		Temp      float64 `json:"temp"`
		Humidity  int     `json:"humidity"`
		FeelsLike float64 `json:"feels_like"`
		Pressure  int     `json:"pressure"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
		Main        string `json:"main"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	Visibility int `json:"visibility"`
}

type WeatherService struct {
	apiKey string
}

func NewWeatherService() *WeatherService {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		panic("OPENWEATHER_API_KEY environment variable is required")
	}
	return &WeatherService{apiKey: apiKey}
}

// GetWeatherData получает данные о погоде для указанного города
func (ws *WeatherService) GetWeatherData(city string) (*WeatherData, error) {
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=ru",
		city, ws.apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error: received status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var weather WeatherData
	err = json.Unmarshal(bodyBytes, &weather)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &weather, nil
}

// FormatWeatherMessage форматирует данные о погоде в красивое сообщение
func (ws *WeatherService) FormatWeatherMessage(weather *WeatherData) string {
	var description string
	if len(weather.Weather) > 0 {
		description = strings.Title(weather.Weather[0].Description)
	} else {
		description = "Нет данных"
	}

	// Эмодзи для разных погодных условий
	emoji := ws.getWeatherEmoji(weather)

	return fmt.Sprintf(`%s Погода в %s:

🌡️ Температура: %.1f°C
💨 Ощущается как: %.1f°C
📊 Давление: %d hPa
💧 Влажность: %d%%
🌬️ Ветер: %.1f м/с
👁️ Видимость: %d км

%s`, emoji, weather.Name, weather.Main.Temp, weather.Main.FeelsLike,
		weather.Main.Pressure, weather.Main.Humidity, weather.Wind.Speed,
		weather.Visibility/1000, description)
}

// getWeatherEmoji возвращает эмодзи в зависимости от погодных условий
func (ws *WeatherService) getWeatherEmoji(weather *WeatherData) string {
	if len(weather.Weather) == 0 {
		return "🌤️"
	}

	main := weather.Weather[0].Main
	switch main {
	case "Clear":
		return "☀️"
	case "Clouds":
		return "☁️"
	case "Rain":
		return "🌧️"
	case "Drizzle":
		return "🌦️"
	case "Thunderstorm":
		return "⛈️"
	case "Snow":
		return "❄️"
	case "Mist", "Fog":
		return "🌫️"
	default:
		return "🌤️"
	}
}

// IsValidCity проверяет, существует ли город (базовая проверка)
func (ws *WeatherService) IsValidCity(city string) bool {
	return len(strings.TrimSpace(city)) > 1
}
