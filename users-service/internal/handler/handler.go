package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"users-service/internal/config"
	"users-service/internal/mail"
	"users-service/internal/model"
	"users-service/internal/security"
	"users-service/internal/store"
)

type Handler struct {
	cfg    *config.Config
	store  *store.Store
	jwt    *security.JWT
	mailer mail.Mailer
	log    *slog.Logger
}

func New(cfg *config.Config, st *store.Store, jwt *security.JWT, mailer mail.Mailer, logger *slog.Logger) *Handler {
	return &Handler{cfg: cfg, store: st, jwt: jwt, mailer: mailer, log: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)

	mux.HandleFunc("POST /api/users/register", h.register)
	mux.HandleFunc("POST /api/users/confirm", h.confirm)
	mux.HandleFunc("POST /api/users/login", h.login)
	mux.HandleFunc("POST /api/users/login/otp", h.verifyOTP)
	mux.HandleFunc("POST /api/users/password/forgot", h.forgotPassword)
	mux.HandleFunc("POST /api/users/password/reset", h.resetPassword)

	mux.Handle("GET /api/users/me", h.auth(http.HandlerFunc(h.me)))
	mux.Handle("POST /api/users/logout", h.auth(http.HandlerFunc(h.logout)))
	mux.Handle("POST /api/users/password/change", h.auth(http.HandlerFunc(h.changePassword)))

	return h.withMiddleware(mux)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "baza nije dostupna")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) newLinkToken(ctx context.Context, userID primitive.ObjectID, typ string, ttl time.Duration) (string, error) {
	raw, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	err = h.store.SaveToken(ctx, &model.OneTimeToken{
		UserID:    userID,
		Type:      typ,
		TokenHash: security.HashToken(h.cfg.JWTSecret, raw),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (h *Handler) clientLink(path, token string) string {
	return fmt.Sprintf("%s%s?token=%s", h.cfg.ClientURL, path, url.QueryEscape(token))
}

func (h *Handler) sendAsync(to, subject, body string) {
	go func() {
		if err := h.mailer.Send(to, subject, body); err != nil {
			h.log.Error("slanje imejla nije uspelo", "error", err.Error())
		}
	}()
}
