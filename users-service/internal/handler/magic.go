package handler

import (
	"errors"
	"net/http"
	"time"

	"users-service/internal/model"
	"users-service/internal/security"
	"users-service/internal/store"
	"users-service/internal/validate"
)

type magicRequestBody struct {
	Email string `json:"email"`
}

func (h *Handler) magicRequest(w http.ResponseWriter, r *http.Request) {
	var req magicRequestBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	email, err := validate.Email(req.Email)
	if err != nil {
		h.rejectInput(w, r, err)
		return
	}

	user, err := h.store.UserByEmail(r.Context(), email)
	if err == nil && user.Active {
		raw, tokErr := h.newLinkToken(r.Context(), user.ID, model.TokenMagic, h.cfg.MagicTTL)
		if tokErr != nil {
			h.log.Error("kreiranje magicnog linka nije uspelo", "error", tokErr.Error())
		} else {
			h.sendAsync(user.Email, "Magicni link za prijavu",
				"Zdravo "+user.FirstName+",\n\n"+
					"Za prijavu bez lozinke otvorite link (vazi "+h.cfg.MagicTTL.String()+", moze se iskoristiti jednom):\n"+
					h.clientLink("/magic-login", raw)+"\n\n"+
					"Ako niste trazili prijavu, ignorisite ovu poruku.\n")
			h.log.Info("zatrazen magicni link", "userId", user.ID.Hex())
		}
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.log.Error("trazenje korisnika nije uspelo", "error", err.Error())
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Ako nalog postoji, poslat je imejl sa magicnim linkom.",
	})
}

type magicLoginBody struct {
	Token string `json:"token"`
}

func (h *Handler) magicLogin(w http.ResponseWriter, r *http.Request) {
	var req magicLoginBody
	if err := decodeJSON(w, r, &req); err != nil || req.Token == "" || len(req.Token) > 200 {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	const invalidLink = "Magicni link nije vazeci ili je istekao"

	ctx := r.Context()
	tok, err := h.store.TokenByHash(ctx, model.TokenMagic, security.HashToken(h.cfg.JWTSecret, req.Token))
	if errors.Is(err, store.ErrNotFound) {
		h.log.Warn("neuspesna prijava: neispravan magicni link", "ip", clientIP(r))
		writeError(w, http.StatusBadRequest, invalidLink)
		return
	}
	if err != nil {
		h.serverError(w, "trazenje magicnog linka", err)
		return
	}
	ok, err := h.store.ConsumeToken(ctx, tok.ID)
	if err != nil {
		h.serverError(w, "iskoriscavanje magicnog linka", err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, invalidLink)
		return
	}

	user, err := h.store.UserByID(ctx, tok.UserID)
	if err != nil || !user.Active {
		writeError(w, http.StatusBadRequest, invalidLink)
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

	expired := time.Now().UTC().After(user.PasswordChangedAt.Add(h.cfg.PasswordMaxAge))
	h.log.Info("uspesna prijava magicnim linkom", "userId", user.ID.Hex(), "ip", clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"token":           token,
		"expiresAt":       exp,
		"user":            h.toDTO(user),
		"passwordExpired": expired,
	})
}
