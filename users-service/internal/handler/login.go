package handler

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"users-service/internal/model"
	"users-service/internal/security"
	"users-service/internal/store"
	"users-service/internal/validate"
)

const msgInvalidCredentials = "Pogresan imejl ili lozinka"

var otpRe = regexp.MustCompile(`^\d{6}$`)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	email, err := validate.Email(req.Email)
	if err != nil || req.Password == "" || len(req.Password) > 72 {
		h.log.Warn("validacija ulaza nije prosla", "path", r.URL.Path, "ip", clientIP(r))
		writeError(w, http.StatusBadRequest, "Neispravan imejl ili lozinka")
		return
	}

	ctx := r.Context()
	user, err := h.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		security.BurnPasswordCheck(req.Password)
		h.log.Warn("neuspesna prijava: nepoznat nalog", "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, msgInvalidCredentials)
		return
	}
	if err != nil {
		h.serverError(w, "trazenje korisnika", err)
		return
	}

	now := time.Now().UTC()
	if now.Before(user.LockedUntil) {
		h.log.Warn("prijava na zakljucan nalog", "userId", user.ID.Hex(), "ip", clientIP(r))
		writeError(w, http.StatusTooManyRequests, "Nalog je privremeno zakljucan zbog previse neuspesnih pokusaja")
		return
	}

	if !security.CheckPassword(user.PasswordHash, req.Password) {
		if err := h.store.RecordFailedLogin(ctx, user.ID, h.cfg.MaxFailedLogins, h.cfg.LockDuration); err != nil {
			h.log.Error("evidencija neuspesne prijave nije uspela", "error", err.Error())
		}
		h.log.Warn("neuspesna prijava: pogresna lozinka", "userId", user.ID.Hex(), "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, msgInvalidCredentials)
		return
	}

	if !user.Active {
		writeErrorCode(w, http.StatusForbidden, "not_confirmed", "Nalog nije potvrdjen. Proverite imejl.")
		return
	}
	if now.After(user.PasswordChangedAt.Add(h.cfg.PasswordMaxAge)) {
		h.log.Warn("prijava sa isteklom lozinkom", "userId", user.ID.Hex())
		writeErrorCode(w, http.StatusForbidden, "password_expired",
			"Lozinka je istekla. Prijava je onemogucena dok je ne resetujete preko imejla.")
		return
	}

	otp, err := security.RandomOTP()
	if err != nil {
		h.serverError(w, "generisanje OTP-a", err)
		return
	}
	err = h.store.SaveToken(ctx, &model.OneTimeToken{
		UserID:    user.ID,
		Type:      model.TokenOTP,
		TokenHash: security.HashToken(h.cfg.JWTSecret, user.ID.Hex()+":"+otp),
		ExpiresAt: now.Add(h.cfg.OTPTTL),
		CreatedAt: now,
	})
	if err != nil {
		h.serverError(w, "upis OTP-a", err)
		return
	}
	body := "Zdravo " + user.FirstName + ",\n\nVas jednokratni kod za prijavu je: " + otp + "\n" +
		"Kod vazi " + h.cfg.OTPTTL.String() + " i moze se iskoristiti samo jednom.\n"
	if err := h.mailer.Send(user.Email, "Jednokratni kod za prijavu", body); err != nil {
		h.log.Error("slanje OTP-a nije uspelo", "userId", user.ID.Hex(), "error", err.Error())
		writeError(w, http.StatusServiceUnavailable, "Slanje koda nije uspelo, pokusajte ponovo")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"otpRequired": true,
		"message":     "Jednokratni kod je poslat na vas imejl.",
	})
}

type otpRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *Handler) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var req otpRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	email, err := validate.Email(req.Email)
	if err != nil || !otpRe.MatchString(req.Code) {
		h.log.Warn("validacija ulaza nije prosla", "path", r.URL.Path, "ip", clientIP(r))
		writeError(w, http.StatusBadRequest, "Neispravan imejl ili kod")
		return
	}
	const invalidCode = "Kod nije ispravan ili je istekao"

	ctx := r.Context()
	user, err := h.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, invalidCode)
		return
	}
	if err != nil {
		h.serverError(w, "trazenje korisnika", err)
		return
	}

	now := time.Now().UTC()
	if now.Before(user.LockedUntil) {
		writeError(w, http.StatusTooManyRequests, "Nalog je privremeno zakljucan zbog previse neuspesnih pokusaja")
		return
	}

	tok, err := h.store.ActiveTokenForUser(ctx, user.ID, model.TokenOTP)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, invalidCode)
		return
	}
	if err != nil {
		h.serverError(w, "trazenje OTP-a", err)
		return
	}

	given := security.HashToken(h.cfg.JWTSecret, user.ID.Hex()+":"+req.Code)
	if !security.Equal(tok.TokenHash, given) {
		attempts, err := h.store.IncrementTokenAttempts(ctx, tok.ID)
		if err == nil && attempts >= h.cfg.MaxOTPAttempts {
			_, _ = h.store.ConsumeToken(ctx, tok.ID)
			_ = h.store.RecordFailedLogin(ctx, user.ID, 1, h.cfg.LockDuration)
		}
		h.log.Warn("neuspesna prijava: pogresan OTP", "userId", user.ID.Hex(), "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, invalidCode)
		return
	}

	ok, err := h.store.ConsumeToken(ctx, tok.ID)
	if err != nil {
		h.serverError(w, "iskoriscavanje OTP-a", err)
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, invalidCode)
		return
	}
	if !user.Active || now.After(user.PasswordChangedAt.Add(h.cfg.PasswordMaxAge)) {
		writeError(w, http.StatusForbidden, "Prijava nije dozvoljena")
		return
	}

	if err := h.store.ResetFailedLogins(ctx, user.ID); err != nil {
		h.log.Error("resetovanje brojaca nije uspelo", "error", err.Error())
	}
	token, _, exp, err := h.jwt.Issue(user.ID.Hex(), user.Role)
	if err != nil {
		h.serverError(w, "izdavanje tokena", err)
		return
	}

	h.log.Info("uspesna prijava", "userId", user.ID.Hex(), "ip", clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"token":     token,
		"expiresAt": exp,
		"user":      h.toDTO(user),
	})
}
