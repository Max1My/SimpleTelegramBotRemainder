package main

import (
	"TelegramBotReminder/application"
	"TelegramBotReminder/domain"
	"TelegramBotReminder/handler"
	"TelegramBotReminder/infrastructure"
	"context"
	"github.com/joho/godotenv"
	"gopkg.in/telebot.v3"
	"log"
	"os"
	"time"
)

func init() {
	log.SetOutput(os.Stdout) // Убедитесь, что логи выводятся в стандартный вывод
}

func main() {
	log.Println("This is a test log message.")
	// Загружаем переменные окружения из файла .env
	if err := godotenv.Load(); err != nil {
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
	client, collection, err := infrastructure.ConnectMongoDB()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Создаем репозиторий и сервис
	reminderRepo := domain.NewMongoReminderRepository(collection)
	reminderService := application.NewReminderService(reminderRepo)

	// Создаем нового бота
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Устанавливаем команды бота
	setBotCommands(bot)

	// Создаем обработчик команд бота
	botHandler := handler.NewReminderHandler(bot, reminderService, botUsername)

	// Запускаем обработчик сообщений
	botHandler.HandleMessages()

	// Фоновый процесс для проверки напоминаний
	go func() {
		for {
			now := time.Now().In(time.FixedZone("UTC+3", 3*60*60))

			dueReminders, err := reminderService.FindDueReminders(context.TODO(), now)
			if err != nil {
				log.Println("Ошибка при поиске напоминаний:", err)
				time.Sleep(1 * time.Minute)
				continue
			}

			for _, reminder := range dueReminders {
				_, err := bot.Send(&telebot.Chat{ID: reminder.ChatID}, "Напоминание: "+reminder.Message)
				if err != nil {
					log.Println("Ошибка при отправке напоминания:", err)
				}
				err = reminderService.DeleteReminder(context.TODO(), reminder.ID, reminder.ChatID)
				if err != nil {
					log.Println("Ошибка при удалении напоминания:", err)
				}
			}

			time.Sleep(1 * time.Minute)
		}
	}()

	bot.Start()
}

func setBotCommands(bot *telebot.Bot) {
	commands := []telebot.Command{
		{Text: "remind", Description: "Установить напоминание"},
		{Text: "list", Description: "Показать список всех напоминаний"},
		{Text: "delete", Description: "Удалить напоминание по ID"},
		{Text: "edit", Description: "Отредактировать напоминание по ID"},
		{Text: "help", Description: "Показать список команд"},
	}

	var cmdInterfaces []interface{}
	for _, cmd := range commands {
		cmdInterfaces = append(cmdInterfaces, cmd)
	}

	if err := bot.SetCommands(cmdInterfaces...); err != nil {
		log.Printf("Ошибка при установке команд: %v", err)
	}
}
