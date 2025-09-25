# GreenAssistantBot

GreenAssistantBot - это умный Telegram-бот, разработанный для помощи пользователям с различными задачами. Бот предоставляет возможность управления профилем, получения информации о погоде и настройки уведомлений.

## 🚀 Возможности

- **👤 Управление профилем**: Создание и редактирование личного профиля с указанием имени и города
- **🌡️ Погода**: Получение актуальной информации о погоде для любого города
- **⚙️ Настройки**: Персонализация работы бота под ваши нужды
- **🔔 Уведомления**: Настройка и получение уведомлений (в разработке)
- **📞 Поддержка**: Получение помощи при использовании бота
- **ℹ️ Информация**: Справка о возможностях бота

## 🛠️ Технологии

- **Язык программирования**: Go
- **База данных**: MySQL с использованием GORM
- **API**: Telegram Bot API, OpenWeatherMap API
- **Архитектура**: Чистая архитектура с разделением на внутренние модули

## 📋 Требования

- Go 1.19 или выше
- MySQL 8.0 или выше
- Токен Telegram-бота
- API-ключ OpenWeatherMap

## 🚀 Установка и запуск

1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/yourusername/GreenAssistantBot.git
   cd GreenAssistantBot
   ```

2. **Создайте файл `.env`** на основе примера `.env.example`:
   ```bash
   cp .env.example .env
   ```

3. **Заполните файл `.env`** необходимыми данными:
   ```bash
   DB_CONNECTION=mysql
   DB_HOST=localhost
   DB_PORT=3306
   DB_DATABASE=green_assistant_bot
   DB_USERNAME=your_username
   DB_PASSWORD=your_password
   
   HTTP_PORT=8585
   
   BOT_TOKEN=your_telegram_bot_token
   BOT_WEBHOOK_URL=https://your-domain.com/webhook
   
   OPENWEATHER_API_KEY=your_openweather_api_key
   ```

4. **Установите зависимости**:
   ```bash
   go mod download
   ```

5. **Запустите миграции базы данных** (при необходимости):
   ```bash
   go run cmd/bot/main.go
   ```
   Миграции будут выполнены автоматически при первом запуске.

6. **Запустите бота**:
   ```bash
   go run cmd/bot/main.go
   ```

## 🏗️ Структура проекта

```
GreenAssistantBot/
├── cmd/                    # Точка входа в приложение
│   └── bot/
│       └── main.go         # Основной файл запуска бота
├── internal/               # Внутренние пакеты приложения
│   ├── bot/                # Логика работы Telegram-бота
│   │   ├── commands.go     # Команды бота
│   │   ├── handlers.go     # Обработчики сообщений
│   │   └── keyboards.go    # Клавиатуры бота
│   ├── database/           # Работа с базой данных
│   │   ├── database.go     # Функции для работы с БД
│   │   └── models/         # Модели данных
│   │       └── user.go     # Модель пользователя
│   ├── storage/            # Хранение данных в памяти
│   │   └── storage.go      # Реализация кэширования
│   └── weather/            # Работа с погодой
│       └── weather.go      # Сервис погоды
├── pkg/                    # Внешние пакеты, которые могут быть использованы в других проектах
│   └── models/             
│       └── model.go        # Базовые модели данных
├── .env                    # Переменные окружения
└── README.md               # Документация проекта
```

## 🤝 Вклад в проект

Мы приветствуем вклад в развитие проекта! Если вы хотите внести свой вклад, пожалуйста:

1. Сделайте форк репозитория
2. Создайте ветку для вашей функции (`git checkout -b feature/amazing-feature`)
3. Внесите изменения и закоммитьте их (`git commit -m 'Add amazing feature'`)
4. Отправьте изменения в ваш форк (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. Подробности см. в файле [LICENSE](LICENSE).

## 📞 Контакты

Если у вас есть вопросы или предложения, пожалуйста, свяжитесь с нами:

- Telegram: [@green_tmz](https://t.me/green_tmz)
- Email: green.tmz@yandex.ru

## 🙏 Благодарности

- [Telegram Bot API](https://core.telegram.org/bots/api) за предоставление отличной платформы для создания ботов
- [OpenWeatherMap](https://openweathermap.org/api) за API погоды
- Сообществу разработчиков Go за вдохновение и помощь