package domain

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type ReminderRepository interface {
	Insert(ctx context.Context, reminder Reminder) error
	FindAll(ctx context.Context, chatID int64) ([]Reminder, error)
	Delete(ctx context.Context, id primitive.ObjectID, chatID int64) error
	Update(ctx context.Context, id primitive.ObjectID, reminder Reminder) error
	FindDueReminders(ctx context.Context, now time.Time) ([]Reminder, error)
}

type MongoReminderRepository struct {
	collection *mongo.Collection
}

func NewMongoReminderRepository(collection *mongo.Collection) *MongoReminderRepository {
	return &MongoReminderRepository{collection: collection}
}

func (r *MongoReminderRepository) Insert(ctx context.Context, reminder Reminder) error {
	_, err := r.collection.InsertOne(ctx, reminder)
	return err
}

func (r *MongoReminderRepository) FindAll(ctx context.Context, chatID int64) ([]Reminder, error) {
	filter := bson.M{"chat_id": chatID}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reminders []Reminder
	if err := cursor.All(ctx, &reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}

func (r *MongoReminderRepository) Delete(ctx context.Context, id primitive.ObjectID, chatID int64) error {
	filter := bson.M{"_id": id, "chat_id": chatID}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

func (r *MongoReminderRepository) Update(ctx context.Context, id primitive.ObjectID, reminder Reminder) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": reminder}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoReminderRepository) FindDueReminders(ctx context.Context, now time.Time) ([]Reminder, error) {
	filter := bson.M{"time": bson.M{"$lte": now}}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reminders []Reminder
	if err := cursor.All(ctx, &reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}
