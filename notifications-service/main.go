package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT nije postavljen")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "notifications servis radi")
	})

	log.Println("Server pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}