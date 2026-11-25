package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"users-service/internal/config"
	"users-service/internal/handler"
	"users-service/internal/mail"
	"users-service/internal/model"
	"users-service/internal/security"
	"users-service/internal/store"
	"users-service/internal/validate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("neispravna konfiguracija", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("konfiguracija ucitana",
		"port", cfg.Port,
		"mongoDb", cfg.MongoDB,
		"smtpHost", cfg.SMTPHost,
		"smtpPort", cfg.SMTPPort,
		"clientUrl", cfg.ClientURL,
		"jwtTtl", cfg.JWTTTL.String(),
		"passwordMaxAge", cfg.PasswordMaxAge.String(),
		"passwordMinAge", cfg.PasswordMinAge.String(),
		"rateLimitPerMin", cfg.RateLimitPerMin,
		"authRateLimitPerMin", cfg.AuthRateLimitPerMin,
		"trustProxy", cfg.TrustProxy,
		"seedDemoUsers", cfg.SeedDemoUsers)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.MongoURI, cfg.MongoDB, logger)
	if err != nil {
		logger.Error("povezivanje sa bazom nije uspelo", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = st.Close(closeCtx)
	}()

	seedAdmin(ctx, cfg, st, logger)
	if cfg.SeedDemoUsers {
		seedDemoUsers(ctx, cfg, st, logger)
	}

	h := handler.New(cfg, st, security.NewJWT(cfg.JWTSecret, cfg.JWTTTL),
		mail.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.MailFrom), logger)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("users servis pokrenut", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server se zaustavio", "error", err.Error())
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info("users servis zaustavljen")
}

func seedAdmin(ctx context.Context, cfg *config.Config, st *store.Store, logger *slog.Logger) {
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return
	}
	email, err := validate.Email(cfg.AdminEmail)
	if err != nil {
		logger.Error("ADMIN_EMAIL nije ispravan")
		return
	}
	if _, err := st.UserByEmail(ctx, email); err == nil {
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		logger.Error("provera administratora nije uspela", "error", err.Error())
		return
	}
	hash, err := security.HashPassword(cfg.AdminPassword)
	if err != nil {
		logger.Error("heshiranje lozinke administratora nije uspelo", "error", err.Error())
		return
	}
	now := time.Now().UTC()
	admin := &model.User{
		FirstName:         "Administrator",
		LastName:          "Sistema",
		Email:             email,
		PasswordHash:      hash,
		Role:              model.RoleAdmin,
		Active:            true,
		QuotaBytes:        cfg.DefaultQuotaBytes,
		PasswordChangedAt: now,
		CreatedAt:         now,
	}
	if err := st.CreateUser(ctx, admin); err != nil {
		logger.Error("kreiranje administratora nije uspelo", "error", err.Error())
		return
	}
	logger.Info("kreiran predefinisani administrator")
}

type demoUser struct {
	first, last, email string
	usedMB             int64
	active             bool
	expired            bool
}

func seedDemoUsers(ctx context.Context, cfg *config.Config, st *store.Store, logger *slog.Logger) {
	hash, err := security.HashPassword(cfg.DemoPassword)
	if err != nil {
		logger.Error("heshiranje demo lozinke nije uspelo", "error", err.Error())
		return
	}

	list := []demoUser{
		{"Mika", "Mikić", "mika@example.com", 120, true, false},
		{"Žika", "Žikić", "zika@example.com", 300, true, false},
		{"Pera", "Perić", "pera@example.com", 850, true, false},
		{"Laza", "Lazić", "laza@example.com", 50, true, false},
		{"Nikola", "Nikolić", "nikola@example.com", 600, true, false},
		{"Jovana", "Jovanović", "jovana@example.com", 200, true, false},
		{"Ana", "Anić", "ana@example.com", 900, true, false},
		{"Marko", "Marković", "marko@example.com", 10, true, false},
		{"Sara", "Sarić", "sara@example.com", 450, true, false},
		{"Ivan", "Ivanović", "ivan@example.com", 700, true, false},
		{"Mira", "Istekla", "istekla@example.com", 0, true, true},
		{"Nepotvrđen", "Nalog", "nepotvrdjen@example.com", 0, false, false},
	}

	now := time.Now().UTC()
	created := 0
	ids := map[string]primitive.ObjectID{}
	for i, d := range list {
		changedAt := now.Add(-cfg.PasswordMinAge - time.Hour)
		if d.expired {
			changedAt = now.Add(-cfg.PasswordMaxAge - time.Hour)
		}
		user := &model.User{
			FirstName:         d.first,
			LastName:          d.last,
			Email:             d.email,
			PasswordHash:      hash,
			Role:              model.RoleUser,
			Active:            d.active,
			QuotaBytes:        cfg.DefaultQuotaBytes,
			UsedBytes:         d.usedMB << 20,
			PasswordChangedAt: changedAt,
			CreatedAt:         now.Add(-time.Duration(len(list)-i) * time.Hour),
		}
		if err := st.CreateUser(ctx, user); err != nil {
			if !errors.Is(err, store.ErrEmailTaken) {
				logger.Error("kreiranje demo korisnika nije uspelo", "error", err.Error())
			}
			continue
		}
		created++
		ids[d.email] = user.ID
	}

	requests := []struct {
		email string
		bytes int64
	}{
		{"mika@example.com", 5 << 30},
		{"pera@example.com", 2 << 30},
		{"ana@example.com", 10 << 30},
	}
	for _, rq := range requests {
		id, ok := ids[rq.email]
		if !ok {
			continue
		}
		_ = st.CreateQuotaRequest(ctx, &model.QuotaRequest{
			UserID:         id,
			UserEmail:      rq.email,
			RequestedBytes: rq.bytes,
			Status:         model.QuotaPending,
			CreatedAt:      now,
		})
	}
	logger.Info("demo korisnici upisani u bazu", "created", created, "total", len(list))
}
