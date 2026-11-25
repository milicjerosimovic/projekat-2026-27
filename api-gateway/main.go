package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func clientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Ograničenje broja zahteva ----------

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
		ip := clientIP(r)
		if !rl.allow(ip) {
			log.Println("Ograničenje zahteva: blokiran klijent", ip)
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			http.Error(w, "previše zahteva, pokušajte kasnije", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Provera JWT tokena ----------

type tokenClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func parseToken(secret []byte, tokenStr string) (*tokenClaims, bool) {
	claims := &tokenClaims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !t.Valid || claims.Subject == "" {
		return nil, false
	}
	return claims, true
}

// Zahtev prolazi samo sa važećim tokenom. ID korisnika uzimamo iz tokena,
// a sve što klijent sam upiše kao userId se prepisuje.
func withAuth(next http.Handler, secret []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Klijent ne sme sam da postavlja ova zaglavlja.
		r.Header.Del("X-User-Id")
		r.Header.Del("X-User-Role")

		tokenStr, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tokenStr == "" {
			log.Println("Neuspela kontrola pristupa: zahtev bez tokena, ip", clientIP(r))
			http.Error(w, "potrebna je prijava", http.StatusUnauthorized)
			return
		}

		claims, valid := parseToken(secret, tokenStr)
		if !valid {
			log.Println("Neuspela kontrola pristupa: nevalidan ili istekao token, ip", clientIP(r))
			http.Error(w, "sesija nije važeća", http.StatusUnauthorized)
			return
		}

		q := r.URL.Query()
		q.Set("userId", claims.Subject)
		r.URL.RawQuery = q.Encode()
		r.Header.Set("X-User-Id", claims.Subject)
		r.Header.Set("X-User-Role", claims.Role)

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := mustEnv("PORT")
	notificationsURL := mustEnv("NOTIFICATIONS_URL")
	corsOrigin := mustEnv("CORS_ORIGIN")
	usersURL := mustEnv("USERS_URL")
	filesURL := mustEnv("FILES_URL")
	rateMax := mustEnvInt("RATE_LIMIT_MAX")
	rateWindow := time.Duration(mustEnvInt("RATE_LIMIT_WINDOW_SECONDS")) * time.Second

	jwtSecret := []byte(mustEnv("JWT_SECRET"))
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET je prekratak (najmanje 32 znaka)")
	}

	notifications := withAuth(http.StripPrefix("/api", proxyTo(notificationsURL)), jwtSecret)
	// Users sam proverava token tamo gde treba (registracija i prijava su javne).
	users := proxyTo(usersURL)

	// Files rute ne znaju za prefiks /api/files, pa ga skidamo.
	files := withAuth(http.StripPrefix("/api/files", proxyTo(filesURL)), jwtSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("api gateway radi"))
	})
	mux.Handle("/api/notifications", notifications)
	mux.Handle("/api/notifications/", notifications)
	mux.Handle("/api/users/", users)
	mux.Handle("/api/files/", files)

	limiter := newRateLimiter(rateMax, rateWindow)
	handler := withCORS(withRateLimit(mux, limiter), corsOrigin)

	log.Println("API gateway pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
