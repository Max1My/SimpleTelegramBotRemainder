package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/tucnak/telebot"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Reminder struct {
	Time    time.Time `bson:"time"`
	Message string    `bson:"message"`
	ChatID  int64     `bson:"chat_id"`
}

var remindersCollection *mongo.Collection

func main() {
	// Загружаем переменные окружения из файла .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	// Получаем токен из переменной окружения
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	// Получаем имя бота (для упоминаний в группе)
	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		log.Fatal("BOT_USERNAME не установлен")
	}

	// Подключение к MongoDB
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Подключение к коллекции напоминаний
	remindersCollection = client.Database("telegram_bot").Collection("reminders")

	// Создаем нового бота
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Обработчик команд
	bot.Handle(telebot.OnText, func(m *telebot.Message) {
		// Проверяем, что бот работает в группе
		if m.Chat.Type != telebot.ChatGroup && m.Chat.Type != telebot.ChatSuperGroup {
			return
		}

		// Проверяем, что команда начинается с /remind
		if !strings.HasPrefix(m.Text, "/remind") {
			return
		}

		// Удаляем упоминание бота в команде (если оно есть)
		command := strings.Replace(m.Text, "@"+botUsername, "", 1)
		command = strings.TrimPrefix(command, "/remind ")
		parts := strings.SplitN(command, " ", 3)

		if len(parts) < 3 {
			bot.Send(m.Chat, "Используйте формат: /remind DD-MM HH:MM Ваше сообщение")
			return
		}

		// Разделяем дату и время
		date := parts[0]
		timePart := parts[1]
		eventMessage := parts[2]

		// Парсинг даты и времени
		var day, month, hour, minute int
		_, err = fmt.Sscanf(date, "%d-%d", &day, &month)
		if err != nil {
			bot.Send(m.Chat, "Используйте формат: /remind DD-MM HH:MM Ваше сообщение")
			return
		}

		_, err = fmt.Sscanf(timePart, "%d:%d", &hour, &minute)
		if err != nil {
			bot.Send(m.Chat, "Используйте формат: /remind DD-MM HH:MM Ваше сообщение")
			return
		}

		// Текущий год
		year := time.Now().Year()

		// Приводим время к UTC+3 (Europe/Moscow)
		location, _ := time.LoadLocation("Europe/Moscow")
		eventTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)

		// Сохраняем напоминание в MongoDB
		reminder := Reminder{
			Time:    eventTime,
			Message: eventMessage,
			ChatID:  m.Chat.ID,
		}
		_, err = remindersCollection.InsertOne(context.TODO(), reminder)
		if err != nil {
			bot.Send(m.Chat, "Ошибка при сохранении напоминания.")
			return
		}

		// Отправляем подтверждение в группу
		bot.Send(m.Chat, fmt.Sprintf("Напоминание установлено на %s", eventTime.Format("02-01 15:04")))
	})

	// Фоновый процесс для проверки напоминаний
	go func() {
		for {
			now := time.Now().In(time.FixedZone("UTC+3", 3*60*60))

			// Поиск напоминаний, которые нужно выполнить
			filter := bson.M{"time": bson.M{"$lte": now}}
			cursor, err := remindersCollection.Find(context.TODO(), filter)
			if err != nil {
				log.Println("Ошибка при поиске напоминаний:", err)
				time.Sleep(1 * time.Minute)
				continue
			}

			var dueReminders []Reminder
			if err = cursor.All(context.TODO(), &dueReminders); err != nil {
				log.Println("Ошибка при чтении напоминаний:", err)
				time.Sleep(1 * time.Minute)
				continue
			}

			// Отправка напоминаний
			for _, reminder := range dueReminders {
				bot.Send(&telebot.Chat{ID: reminder.ChatID}, "Напоминание: "+reminder.Message)

				// Удаление отправленных напоминаний
				_, err := remindersCollection.DeleteOne(context.TODO(), bson.M{"_id": reminder.ChatID})
				if err != nil {
					log.Println("Ошибка при удалении напоминания:", err)
				}
			}

			time.Sleep(1 * time.Minute)
		}
	}()

	// Запуск бота
	bot.Start()
}
