package application

import (
	"TelegramBotReminder/domain"
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type ReminderService struct {
	repository domain.ReminderRepository
}

func NewReminderService(repository domain.ReminderRepository) *ReminderService {
	return &ReminderService{repository: repository}
}

func (s *ReminderService) CreateReminder(ctx context.Context, reminderTime time.Time, message string, chatID int64) error {
	reminder := domain.NewReminder(reminderTime, message, chatID)
	err := s.repository.Insert(ctx, reminder)
	if err != nil {
		return err
	}

	// Создание дополнительных напоминаний
	now := time.Now()
	if reminderTime.Sub(now) > 2*time.Hour {
		// Напоминание за сутки до
		reminder24h := domain.NewReminder(reminderTime.Add(-24*time.Hour), "Через 24 часа: "+message, chatID)
		err = s.repository.Insert(ctx, reminder24h)
		if err != nil {
			return err
		}

		// Напоминание за час до
		reminder1h := domain.NewReminder(reminderTime.Add(-1*time.Hour), "Через 1 час: "+message, chatID)
		err = s.repository.Insert(ctx, reminder1h)
		if err != nil {
			return err
		}

		// Напоминание за 5 минут до
		reminder5m := domain.NewReminder(reminderTime.Add(-5*time.Minute), "Через 5 минут: "+message, chatID)
		err = s.repository.Insert(ctx, reminder5m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ReminderService) GetPendingReminders() ([]domain.Reminder, error) {
	return s.repository.FindDueReminders(context.Background(), time.Now())
}

func (s *ReminderService) GetReminders(ctx context.Context, chatID int64) ([]domain.Reminder, error) {
	return s.repository.FindAll(ctx, chatID)
}

func (s *ReminderService) DeleteReminder(ctx context.Context, id primitive.ObjectID, chatID int64) error {
	return s.repository.Delete(ctx, id, chatID)
}

func (s *ReminderService) EditReminder(ctx context.Context, id primitive.ObjectID, time time.Time, message string) error {
	reminder := domain.Reminder{
		ID:      id,
		Time:    time,
		Message: message,
	}
	return s.repository.Update(ctx, id, reminder)
}

func (s *ReminderService) FindDueReminders(ctx context.Context, now time.Time) ([]domain.Reminder, error) {
	return s.repository.FindDueReminders(ctx, now)
}
