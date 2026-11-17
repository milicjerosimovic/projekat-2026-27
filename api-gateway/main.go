package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal(key + " nije postavljen")
	}
	return v
}

func proxyTo(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func withCORS(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := mustEnv("PORT")
	notificationsURL := mustEnv("NOTIFICATIONS_URL")
	corsOrigin := mustEnv("CORS_ORIGIN")

	notifications := http.StripPrefix("/api", proxyTo(notificationsURL))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("api gateway radi"))
	})
	mux.Handle("/api/notifications", notifications)
	mux.Handle("/api/notifications/", notifications)

	log.Println("API gateway pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, withCORS(mux, corsOrigin)))
}
