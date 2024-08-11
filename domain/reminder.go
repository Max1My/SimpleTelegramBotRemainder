package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Reminder struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Time    time.Time          `bson:"time"`
	Message string             `bson:"message"`
	ChatID  int64              `bson:"chat_id"`
}

func NewReminder(time time.Time, message string, chatID int64) Reminder {
	return Reminder{
		ID:      primitive.NewObjectID(),
		Time:    time,
		Message: message,
		ChatID:  chatID,
	}
}
