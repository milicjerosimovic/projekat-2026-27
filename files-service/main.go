package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"folder.go/internal/handler"
	"folder.go/internal/repository"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27018"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)

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

	db := client.Database("filesdb")

	folderRepository := repository.NewFolderRepository(db)
	fileRepository := repository.NewFileRepository(db)

	usersURL := os.Getenv("USERS_SERVICE_URL")
	if usersURL == "" {
		usersURL = "http://localhost:8080"
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./storage"
	}

	h := &handler.Handler{
		Folders:    folderRepository,
		Files:      fileRepository,
		UsersURL:   usersURL,
		StorageDir: storageDir,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Println("Files service is running on port " + port)

	mux := http.NewServeMux()

	mux.Handle("/ui/", http.StripPrefix(
		"/ui/",
		http.FileServer(http.Dir("../frontend/files")),
	))

	mux.Handle("/", h.Routes())

	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}
