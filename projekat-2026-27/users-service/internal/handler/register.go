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

type registerRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}

	first, err := validate.Name("Ime", req.FirstName)
	if err != nil {
		h.rejectInput(w, r, err)
		return
	}
	last, err := validate.Name("Prezime", req.LastName)
	if err != nil {
		h.rejectInput(w, r, err)
		return
	}
	email, err := validate.Email(req.Email)
	if err != nil {
		h.rejectInput(w, r, err)
		return
	}
	if req.Password != req.ConfirmPassword {
		h.rejectInput(w, r, errors.New("Lozinka i ponovljena lozinka se ne poklapaju"))
		return
	}
	if err := validate.Password(req.Password, email); err != nil {
		h.rejectInput(w, r, err)
		return
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		h.serverError(w, "heshiranje lozinke", err)
		return
	}

	now := time.Now().UTC()
	user := &model.User{
		FirstName:         first,
		LastName:          last,
		Email:             email,
		PasswordHash:      hash,
		Role:              model.RoleUser,
		Active:            false,
		QuotaBytes:        h.cfg.DefaultQuotaBytes,
		PasswordChangedAt: now,
		CreatedAt:         now,
	}
	if err := h.store.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "Nalog sa ovom imejl adresom vec postoji")
			return
		}
		h.serverError(w, "upis korisnika", err)
		return
	}

	raw, err := h.newLinkToken(r.Context(), user.ID, model.TokenConfirm, h.cfg.ConfirmTTL)
	if err != nil {
		h.serverError(w, "kreiranje tokena za potvrdu", err)
		return
	}
	h.sendAsync(user.Email, "Potvrda registracije",
		"Zdravo "+user.FirstName+",\n\n"+
			"Za aktivaciju naloga otvorite sledeci link (vazi "+h.cfg.ConfirmTTL.String()+"):\n"+
			h.clientLink("/confirm", raw)+"\n\n"+
			"Ako se niste registrovali, ignorisite ovu poruku.\n")

	h.log.Info("registracija", "userId", user.ID.Hex())
	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "Registracija je uspesna. Proverite imejl i potvrdite nalog.",
	})
}

type confirmRequest struct {
	Token string `json:"token"`
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	if err := decodeJSON(w, r, &req); err != nil || req.Token == "" || len(req.Token) > 200 {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}

	tok, err := h.store.TokenByHash(r.Context(), model.TokenConfirm, security.HashToken(h.cfg.JWTSecret, req.Token))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "Link za potvrdu nije vazeci ili je istekao")
		return
	}
	if err != nil {
		h.serverError(w, "trazenje tokena za potvrdu", err)
		return
	}
	ok, err := h.store.ConsumeToken(r.Context(), tok.ID)
	if err != nil {
		h.serverError(w, "iskoriscavanje tokena", err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "Link za potvrdu nije vazeci ili je istekao")
		return
	}
	if err := h.store.ActivateUser(r.Context(), tok.UserID); err != nil {
		h.serverError(w, "aktivacija naloga", err)
		return
	}

	h.log.Info("nalog potvrdjen", "userId", tok.UserID.Hex())
	writeJSON(w, http.StatusOK, map[string]string{"message": "Nalog je aktiviran. Mozete da se prijavite."})
}

func (h *Handler) rejectInput(w http.ResponseWriter, r *http.Request, err error) {
	h.log.Warn("validacija ulaza nije prosla", "path", r.URL.Path, "ip", clientIP(r))
	writeError(w, http.StatusBadRequest, err.Error())
}

func (h *Handler) serverError(w http.ResponseWriter, what string, err error) {
	h.log.Error("greska servera", "operation", what, "error", err.Error())
	writeError(w, http.StatusInternalServerError, "Doslo je do greske")
}
