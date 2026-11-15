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

type changePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword"`
	NewPassword        string `json:"newPassword"`
	ConfirmNewPassword string `json:"confirmNewPassword"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	user := authFrom(r).User
	claims := authFrom(r).Claims

	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	if len(req.CurrentPassword) > 72 {
		h.rejectInput(w, r, errors.New("Trenutna lozinka nije ispravna"))
		return
	}
	if !security.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		if err := h.store.RecordFailedLogin(r.Context(), user.ID, h.cfg.MaxFailedLogins, h.cfg.LockDuration); err != nil {
			h.log.Error("evidencija neuspesnog pokusaja nije uspela", "error", err.Error())
		}
		h.log.Warn("promena lozinke: pogresna trenutna lozinka", "userId", user.ID.Hex(), "ip", clientIP(r))
		writeError(w, http.StatusForbidden, "Trenutna lozinka nije ispravna")
		return
	}
	if req.NewPassword != req.ConfirmNewPassword {
		h.rejectInput(w, r, errors.New("Nova lozinka i ponovljena lozinka se ne poklapaju"))
		return
	}

	now := time.Now().UTC()
	age := now.Sub(user.PasswordChangedAt)
	if age < h.cfg.PasswordMinAge {
		remaining := (h.cfg.PasswordMinAge - age).Round(time.Second)
		h.log.Warn("promena lozinke odbijena: lozinka je previse nova", "userId", user.ID.Hex())
		writeErrorCode(w, http.StatusForbidden, "password_too_new",
			"Lozinka se moze promeniti tek kada je stara bar "+h.cfg.PasswordMinAge.String()+
				". Preostalo vreme: "+remaining.String())
		return
	}

	if err := validate.Password(req.NewPassword, user.Email); err != nil {
		h.rejectInput(w, r, err)
		return
	}
	if security.CheckPassword(user.PasswordHash, req.NewPassword) {
		h.rejectInput(w, r, errors.New("Nova lozinka mora biti razlicita od trenutne"))
		return
	}

	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		h.serverError(w, "heshiranje lozinke", err)
		return
	}
	if err := h.store.SetPassword(r.Context(), user.ID, hash, now); err != nil {
		h.serverError(w, "promena lozinke", err)
		return
	}
	if claims.ExpiresAt != nil {
		_ = h.store.RevokeJTI(r.Context(), claims.ID, claims.ExpiresAt.Time)
	}

	h.sendAsync(user.Email, "Lozinka je promenjena",
		"Zdravo "+user.FirstName+",\n\nVasa lozinka je upravo promenjena. "+
			"Ako to niste vi, odmah zatrazite reset lozinke.\n")
	h.log.Info("lozinka promenjena", "userId", user.ID.Hex())
	writeJSON(w, http.StatusOK, map[string]string{"message": "Lozinka je promenjena. Prijavite se ponovo."})
}

type forgotRequest struct {
	Email string `json:"email"`
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotRequest
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
		raw, tokErr := h.newLinkToken(r.Context(), user.ID, model.TokenReset, h.cfg.ResetTTL)
		if tokErr != nil {
			h.log.Error("kreiranje tokena za reset nije uspelo", "error", tokErr.Error())
		} else {
			h.sendAsync(user.Email, "Reset lozinke",
				"Zdravo "+user.FirstName+",\n\n"+
					"Za postavljanje nove lozinke otvorite link (vazi "+h.cfg.ResetTTL.String()+"):\n"+
					h.clientLink("/reset-password", raw)+"\n\n"+
					"Ako niste trazili reset, ignorisite ovu poruku.\n")
		}
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.log.Error("trazenje korisnika nije uspelo", "error", err.Error())
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Ako nalog postoji, poslat je imejl sa linkom za reset lozinke.",
	})
}

type resetRequest struct {
	Token              string `json:"token"`
	NewPassword        string `json:"newPassword"`
	ConfirmNewPassword string `json:"confirmNewPassword"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := decodeJSON(w, r, &req); err != nil || req.Token == "" || len(req.Token) > 200 {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	const invalidLink = "Link za reset nije vazeci ili je istekao"

	tok, err := h.store.TokenByHash(r.Context(), model.TokenReset, security.HashToken(h.cfg.JWTSecret, req.Token))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, invalidLink)
		return
	}
	if err != nil {
		h.serverError(w, "trazenje tokena za reset", err)
		return
	}
	user, err := h.store.UserByID(r.Context(), tok.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, invalidLink)
		return
	}

	if req.NewPassword != req.ConfirmNewPassword {
		h.rejectInput(w, r, errors.New("Nova lozinka i ponovljena lozinka se ne poklapaju"))
		return
	}
	if err := validate.Password(req.NewPassword, user.Email); err != nil {
		h.rejectInput(w, r, err)
		return
	}
	if security.CheckPassword(user.PasswordHash, req.NewPassword) {
		h.rejectInput(w, r, errors.New("Nova lozinka mora biti razlicita od trenutne"))
		return
	}

	ok, err := h.store.ConsumeToken(r.Context(), tok.ID)
	if err != nil {
		h.serverError(w, "iskoriscavanje tokena", err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, invalidLink)
		return
	}

	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		h.serverError(w, "heshiranje lozinke", err)
		return
	}
	if err := h.store.SetPassword(r.Context(), user.ID, hash, time.Now().UTC()); err != nil {
		h.serverError(w, "reset lozinke", err)
		return
	}

	h.sendAsync(user.Email, "Lozinka je promenjena",
		"Zdravo "+user.FirstName+",\n\nVasa lozinka je resetovana. "+
			"Ako to niste vi, odmah kontaktirajte administratora.\n")
	h.log.Info("lozinka resetovana", "userId", user.ID.Hex())
	writeJSON(w, http.StatusOK, map[string]string{"message": "Lozinka je promenjena. Mozete da se prijavite."})
}
