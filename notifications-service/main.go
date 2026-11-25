package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

var validUserID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal(key + " nije postavljen")
	}
	return v
}

// Povezuje se na Cassandru (čeka dok baza ne bude spremna),
// pravi keyspace i tabelu ako ne postoje.
func connect(host string, port int, keyspace string) *gocql.Session {
	var admin *gocql.Session
	for attempt := 1; attempt <= 30; attempt++ {
		cluster := gocql.NewCluster(host)
		cluster.Port = port
		cluster.Timeout = 10 * time.Second
		cluster.DisableInitialHostLookup = true
		s, err := cluster.CreateSession()
		if err == nil {
			admin = s
			break
		}
		log.Printf("Cassandra nije spremna (pokušaj %d/30): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
	if admin == nil {
		log.Fatal("Ne mogu da se povežem na Cassandru")
	}

	err := admin.Query(fmt.Sprintf(
		`CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`,
		keyspace)).Exec()
	if err != nil {
		log.Fatal("Greška pri pravljenju keyspace-a: ", err)
	}
	admin.Close()

	cluster := gocql.NewCluster(host)
	cluster.Port = port
	cluster.Keyspace = keyspace
	cluster.Timeout = 10 * time.Second
	cluster.DisableInitialHostLookup = true
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal("Greška pri povezivanju na keyspace: ", err)
	}

	err = session.Query(`CREATE TABLE IF NOT EXISTS notifications (
		user_id text,
		created_at timestamp,
		id uuid,
		type text,
		message text,
		PRIMARY KEY ((user_id), created_at, id)
	) WITH CLUSTERING ORDER BY (created_at DESC, id ASC)`).Exec()
	if err != nil {
		log.Fatal("Greška pri pravljenju tabele: ", err)
	}
	return session
}

// Ubacuje probne notifikacije ako je tabela prazna.
func seed(session *gocql.Session) {
	var count int
	if err := session.Query(`SELECT COUNT(*) FROM notifications`).Scan(&count); err != nil {
		log.Println("Ne mogu da proverim tabelu:", err)
		return
	}
	if count > 0 {
		return
	}

	samples := []Notification{
		{UserID: "user1", Type: "SHARE", Message: "Marko je podelio fajl izvestaj.pdf sa tobom", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{UserID: "user1", Type: "NEW_FILE", Message: "Dodata nova datoteka u folder Projekat", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{UserID: "user2", Type: "NEW_VERSION", Message: "Nova verzija datoteke slika.png", CreatedAt: time.Now()},
	}
	for _, n := range samples {
		err := session.Query(
			`INSERT INTO notifications (user_id, created_at, id, type, message) VALUES (?, ?, ?, ?, ?)`,
			n.UserID, n.CreatedAt, gocql.TimeUUID(), n.Type, n.Message).Exec()
		if err != nil {
			log.Println("Greška pri ubacivanju:", err)
		}
	}
	log.Println("Ubačene probne notifikacije")
}

func main() {
	port := mustEnv("PORT")
	cassandraHost := mustEnv("CASSANDRA_HOST")
	cassandraPort, err := strconv.Atoi(mustEnv("CASSANDRA_PORT"))
	if err != nil {
		log.Fatal("CASSANDRA_PORT mora biti broj")
	}
	keyspace := mustEnv("CASSANDRA_KEYSPACE")

	session := connect(cassandraHost, cassandraPort, keyspace)
	defer session.Close()
	log.Println("Povezano sa Cassandrom")
	seed(session)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("notifications servis radi"))
	})

	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metoda nije dozvoljena", http.StatusMethodNotAllowed)
			return
		}

		userID := strings.TrimSpace(r.URL.Query().Get("userId"))
		if !validUserID.MatchString(userID) {
			log.Println("Neuspela validacija: nevalidan userId")
			http.Error(w, "userId nije validan", http.StatusBadRequest)
			return
		}

		iter := session.Query(
			`SELECT id, user_id, type, message, created_at FROM notifications WHERE user_id = ?`,
			userID).Iter()

		result := []Notification{}
		var id gocql.UUID
		var uid, typ, msg string
		var created time.Time
		for iter.Scan(&id, &uid, &typ, &msg, &created) {
			result = append(result, Notification{
				ID: id.String(), UserID: uid, Type: typ, Message: msg, CreatedAt: created,
			})
		}
		if err := iter.Close(); err != nil {
			log.Println("Greška pri čitanju iz baze:", err)
			http.Error(w, "greška servera", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Println("Server pokrenut na portu", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
