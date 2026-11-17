package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"users-service/internal/model"
	"users-service/internal/security"
	"users-service/internal/store"
)

type ctxKey struct{}

type authInfo struct {
	User   *model.User
	Claims *security.Claims
}

func authFrom(r *http.Request) *authInfo {
	info, _ := r.Context().Value(ctxKey{}).(*authInfo)
	return info
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (h *Handler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if p := recover(); p != nil {
				h.log.Error("panika u rukovaocu", "panic", fmt.Sprint(p))
				writeError(rec, http.StatusInternalServerError, "Doslo je do greske")
			}
			h.log.Info("zahtev", "method", r.Method, "path", r.URL.Path, "status", rec.status,
				"durationMs", time.Since(start).Milliseconds(), "ip", clientIP(r))
		}()

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(rec, r)
	})
}

func (h *Handler) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tokenStr == "" {
			h.log.Warn("pristup bez tokena", "path", r.URL.Path, "ip", clientIP(r))
			writeError(w, http.StatusUnauthorized, "Potrebna je prijava")
			return
		}
		claims, err := h.jwt.Parse(tokenStr)
		if err != nil {
			h.log.Warn("nevalidan ili istekao token", "path", r.URL.Path, "ip", clientIP(r))
			writeError(w, http.StatusUnauthorized, "Sesija nije vazeca")
			return
		}

		revoked, err := h.store.IsRevoked(r.Context(), claims.ID)
		if err != nil {
			h.log.Error("provera opozvanog tokena nije uspela", "error", err.Error())
			writeError(w, http.StatusInternalServerError, "Doslo je do greske")
			return
		}
		if revoked {
			h.log.Warn("pokusaj upotrebe opozvanog tokena", "ip", clientIP(r))
			writeError(w, http.StatusUnauthorized, "Sesija nije vazeca")
			return
		}

		uid, err := primitive.ObjectIDFromHex(claims.Subject)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Sesija nije vazeca")
			return
		}
		user, err := h.store.UserByID(r.Context(), uid)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "Sesija nije vazeca")
			return
		}
		if err != nil {
			h.log.Error("ucitavanje korisnika nije uspelo", "error", err.Error())
			writeError(w, http.StatusInternalServerError, "Doslo je do greske")
			return
		}
		if !user.Active ||
			(claims.IssuedAt != nil && claims.IssuedAt.Time.Before(user.TokensValidAfter.Truncate(time.Second))) {
			writeError(w, http.StatusUnauthorized, "Sesija nije vazeca")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKey{}, &authInfo{User: user, Claims: claims})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
