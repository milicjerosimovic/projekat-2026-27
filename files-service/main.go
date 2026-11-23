
package main

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        mongoURI = "mongodb://localhost:27018"
    }

    clientOptions := options.Client()
    clientOptions.ApplyURI(mongoURI)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatal("MongoDB connection error: ", err)
    }

    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal("MongoDB ping error: ", err)
    }

    defer func() {
        disconnectCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
        defer stop()
        if err := client.Disconnect(disconnectCtx); err != nil {
            log.Println("MongoDB disconnect error: ", err)
        }
    }()

    log.Println("Successfully connected to MongoDB!")
}

