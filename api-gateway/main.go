package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal(key + " nije postavljen")
	}
	return v
}

func mustEnvInt(key string) int {
	n, err := strconv.Atoi(mustEnv(key))
	if err != nil || n <= 0 {
		log.Fatal(key + " mora biti pozitivan ceo broj")
	}
	return n
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

// Ograničenje broja zahteva: najviše `max` zahteva po klijentu u prozoru od `window`.
type visitor struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{visitors: map[string]*visitor{}, max: max, window: window}
	// Povremeno čistimo stare zapise da memorija ne raste.
	go func() {
		for range time.Tick(window) {
			rl.mu.Lock()
			for key, v := range rl.visitors {
				if time.Since(v.windowStart) > window {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok || time.Since(v.windowStart) > rl.window {
		rl.visitors[key] = &visitor{count: 1, windowStart: time.Now()}
		return true
	}
	v.count++
	return v.count <= rl.max
}

func withRateLimit(next http.Handler, rl *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !rl.allow(ip) {
			log.Println("Ograničenje zahteva: blokiran klijent", ip)
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			http.Error(w, "previše zahteva, pokušajte kasnije", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := mustEnv("PORT")
	notificationsURL := mustEnv("NOTIFICATIONS_URL")
	corsOrigin := mustEnv("CORS_ORIGIN")
	rateMax := mustEnvInt("RATE_LIMIT_MAX")
	rateWindow := time.Duration(mustEnvInt("RATE_LIMIT_WINDOW_SECONDS")) * time.Second

	notifications := http.StripPrefix("/api", proxyTo(notificationsURL))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("api gateway radi"))
	})
	mux.Handle("/api/notifications", notifications)
	mux.Handle("/api/notifications/", notifications)

	limiter := newRateLimiter(rateMax, rateWindow)
	handler := withCORS(withRateLimit(mux, limiter), corsOrigin)

	log.Println("API gateway pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
