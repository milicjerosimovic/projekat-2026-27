package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

var notifications = []Notification{
	{ID: "1", UserID: "user1", Type: "SHARE", Message: "Marko je podelio fajl izvestaj.pdf sa tobom", CreatedAt: time.Now().Add(-2 * time.Hour)},
	{ID: "2", UserID: "user1", Type: "NEW_FILE", Message: "Dodata nova datoteka u folder Projekat", CreatedAt: time.Now().Add(-1 * time.Hour)},
	{ID: "3", UserID: "user2", Type: "NEW_VERSION", Message: "Nova verzija datoteke slika.png", CreatedAt: time.Now()},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT nije postavljen")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("notifications servis radi"))
	})

	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("userId")
		result := []Notification{}
		for _, n := range notifications {
			if userID == "" || n.UserID == userID {
				result = append(result, n)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Println("Server pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
