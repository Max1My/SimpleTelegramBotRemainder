package infrastructure

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

func ConnectMongoDB() (*mongo.Client, *mongo.Collection, error) {
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, nil, err
	}

	collection := client.Database("telegram_bot").Collection("reminders")
	return client, collection, nil
}
