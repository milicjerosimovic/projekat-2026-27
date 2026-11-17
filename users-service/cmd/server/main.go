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
