package handler

import (
	"TelegramBotReminder/application"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/telebot.v3"
	"log"
	"strings"
	"time"
)

type ReminderHandler struct {
	bot             *telebot.Bot
	reminderService *application.ReminderService
	botUsername     string
}

func NewReminderHandler(bot *telebot.Bot, reminderService *application.ReminderService, botUsername string) *ReminderHandler {
	return &ReminderHandler{
		bot:             bot,
		reminderService: reminderService,
		botUsername:     botUsername,
	}
}

func (h *ReminderHandler) HandleMessages() {
	h.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		log.Println("Received a message") // Проверьте, что это сообщение выводится
		m := c.Message()
		if m.Chat.Type != telebot.ChatGroup && m.Chat.Type != telebot.ChatSuperGroup {
			return nil
		}

		switch {
		case strings.HasPrefix(m.Text, "/remind"):
			return h.handleRemindCommand(c)
		case strings.HasPrefix(m.Text, "/list"):
			return h.handleListCommand(c)
		case strings.HasPrefix(m.Text, "/delete"):
			return h.handleDeleteCommand(c)
		case strings.HasPrefix(m.Text, "/edit"):
			return h.handleEditCommand(c)
		case strings.HasPrefix(m.Text, "/help"):
			return h.handleHelpCommand(c)
		}
		return nil
	})
}

func (h *ReminderHandler) handleRemindCommand(c telebot.Context) error {
	m := c.Message()
	command := strings.Replace(m.Text, "@"+h.botUsername, "", 1)
	command = strings.TrimPrefix(command, "/remind ")
	parts := strings.Fields(command)

	if len(parts) < 3 {
		return c.Send("Используйте формат: /remind DD-MM HH:MM Ваше сообщение или /remind через X минут/часов/дней Ваше сообщение")
	}

	var eventTime time.Time
	var err error

	if parts[0] == "через" {
		durationStr := strings.Join(parts[1:3], " ")
		eventMessage := strings.Join(parts[3:], " ")
		duration, err := parseDuration(durationStr)
		if err != nil {
			return c.Send("Неверный формат времени. Используйте, например: через 20 минут, через 2 часа.")
		}
		fmt.Println(duration)

		eventTime = time.Now().Add(duration)
		err = h.reminderService.CreateReminder(context.Background(), eventTime, eventMessage, m.Chat.ID)
	} else {
		date := parts[0]
		timePart := parts[1]
		eventMessage := strings.Join(parts[2:], " ")

		var day, month, hour, minute int
		_, err = fmt.Sscanf(date, "%d-%d", &day, &month)
		if err != nil {
			return c.Send("Используйте формат: /remind DD-MM HH:MM Ваше сообщение")
		}

		_, err = fmt.Sscanf(timePart, "%d:%d", &hour, &minute)
		if err != nil {
			return c.Send("Используйте формат: /remind DD-MM HH:MM Ваше сообщение")
		}

		year := time.Now().Year()
		location, _ := time.LoadLocation("Europe/Moscow")
		eventTime = time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)

		err = h.reminderService.CreateReminder(context.Background(), eventTime, eventMessage, m.Chat.ID)
	}

	if err != nil {
		return c.Send("Ошибка при создании напоминания.")
	}

	return c.Send("Успешно добавление напоминания")
}

func (h *ReminderHandler) handleListCommand(c telebot.Context) error {
	m := c.Message()
	reminders, err := h.reminderService.GetReminders(context.Background(), m.Chat.ID)
	if err != nil {
		return c.Send("Ошибка при получении напоминаний.")
	}

	if len(reminders) == 0 {
		return c.Send("Нет предстоящих напоминаний.")
	}

	message := "Список предстоящих напоминаний:\n"
	for _, reminder := range reminders {
		message += fmt.Sprintf("ID: %s, %s: %s\n", reminder.ID.Hex(), reminder.Time.Format("02-01 15:04"), reminder.Message)
	}

	return c.Send(message)
}

func (h *ReminderHandler) handleDeleteCommand(c telebot.Context) error {
	m := c.Message()
	command := strings.TrimPrefix(m.Text, "/delete ")
	if command == "" {
		return c.Send("Используйте формат: /delete ID")
	}

	id, err := primitive.ObjectIDFromHex(command)
	if err != nil {
		return c.Send("Неверный формат ID.")
	}

	err = h.reminderService.DeleteReminder(context.Background(), id, m.Chat.ID)
	if err != nil {
		return c.Send("Ошибка при удалении напоминания.")
	}

	return c.Send("Успешное удаление напоминания")
}

func (h *ReminderHandler) handleEditCommand(c telebot.Context) error {
	m := c.Message()
	command := strings.TrimPrefix(m.Text, "/edit ")
	parts := strings.SplitN(command, " ", 4)

	if len(parts) < 4 {
		return c.Send("Используйте формат: /edit ID DD-MM HH:MM Ваше сообщение")
	}

	idStr := parts[0]
	date := parts[1]
	timePart := parts[2]
	eventMessage := parts[3]

	var day, month, hour, minute int
	_, err := fmt.Sscanf(date, "%d-%d", &day, &month)
	if err != nil {
		return c.Send("Используйте формат: /edit ID DD-MM HH:MM Ваше сообщение")
	}

	_, err = fmt.Sscanf(timePart, "%d:%d", &hour, &minute)
	if err != nil {
		return c.Send("Используйте формат: /edit ID DD-MM HH:MM Ваше сообщение")
	}

	year := time.Now().Year()
	location, _ := time.LoadLocation("Europe/Moscow")
	eventTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.Send("Неверный формат ID.")
	}

	err = h.reminderService.EditReminder(context.Background(), id, eventTime, eventMessage)
	if err != nil {
		return c.Send("Ошибка при редактировании напоминания.")
	}

	return c.Send("Успешное редактирование напоминания")
}

func (h *ReminderHandler) processReminders() {
	// Получение всех напоминаний, которые должны быть выполнены
	reminders, err := h.reminderService.GetPendingReminders()
	if err != nil {
		log.Println("Ошибка при получении напоминаний:", err)
		return
	}

	for _, reminder := range reminders {
		if time.Now().After(reminder.Time) {
			// Отправка сообщения
			chat := &telebot.Chat{ID: reminder.ChatID}
			_, err := h.bot.Send(chat, reminder.Message)
			if err != nil {
				log.Println("Ошибка при отправке сообщения:", err)
			}

			// Удаление выполненного напоминания
			err = h.reminderService.DeleteReminder(context.Background(), reminder.ID, reminder.ChatID)
			if err != nil {
				log.Println("Ошибка при удалении напоминания:", err)
			}
		}
	}
}

func (h *ReminderHandler) handleHelpCommand(c telebot.Context) error {
	helpMessage := `Доступные команды:
    /remind DD-MM HH:MM Ваше сообщение - Установить напоминание
    /remind через (1, 60) (минут, часов, секунд) Ваше сообщение - Установить напоминание
    /list - Показать список всех напоминаний
    /delete ID - Удалить напоминание по ID
    /edit ID DD-MM HH:MM Ваше сообщение - Отредактировать напоминание по ID
    /help - Показать список команд`

	return c.Send(helpMessage)
}

func parseDuration(input string) (time.Duration, error) {
	var value int
	var unit string

	// Преобразуем строку к нижнему регистру, чтобы избежать проблем с регистрами
	input = strings.ToLower(input)
	_, err := fmt.Sscanf(input, "%d %s", &value, &unit)
	if err != nil {
		return 0, fmt.Errorf("не могу разобрать время: %s", input)
	}

	// Проверка единицы времени и возврат соответствующей продолжительности
	switch unit {
	case "минут", "минута", "минуты", "минуту":
		return time.Duration(value) * time.Minute, nil
	case "час", "часа", "часов":
		return time.Duration(value) * time.Hour, nil
	case "день", "дня", "дней":
		return time.Duration(value) * time.Hour * 24, nil
	default:
		return 0, fmt.Errorf("неизвестная единица времени: %s", unit)
	}
}
