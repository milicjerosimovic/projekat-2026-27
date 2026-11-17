package handler

import (
	"net/http"
	"time"

	"users-service/internal/model"
)

type userDTO struct {
	ID                string    `json:"id"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Email             string    `json:"email"`
	Role              string    `json:"role"`
	QuotaBytes        int64     `json:"quotaBytes"`
	UsedBytes         int64     `json:"usedBytes"`
	PasswordExpiresAt time.Time `json:"passwordExpiresAt"`
}

func (h *Handler) toDTO(u *model.User) userDTO {
	return userDTO{
		ID:                u.ID.Hex(),
		FirstName:         u.FirstName,
		LastName:          u.LastName,
		Email:             u.Email,
		Role:              u.Role,
		QuotaBytes:        u.QuotaBytes,
		UsedBytes:         u.UsedBytes,
		PasswordExpiresAt: u.PasswordChangedAt.Add(h.cfg.PasswordMaxAge),
	}
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.toDTO(authFrom(r).User))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	info := authFrom(r)
	if info.Claims.ExpiresAt != nil {
		if err := h.store.RevokeJTI(r.Context(), info.Claims.ID, info.Claims.ExpiresAt.Time); err != nil {
			h.serverError(w, "odjava", err)
			return
		}
	}
	h.log.Info("odjava", "userId", info.User.ID.Hex())
	w.WriteHeader(http.StatusNoContent)
}
