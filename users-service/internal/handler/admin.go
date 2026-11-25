package handler

import (
	"net/http"
	"time"
)

type adminUserDTO struct {
	ID         string    `json:"id"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Active     bool      `json:"active"`
	QuotaBytes int64     `json:"quotaBytes"`
	UsedBytes  int64     `json:"usedBytes"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (h *Handler) adminUsers(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1, 1, 1000000)
	size := queryInt(r, "size", 20, 1, 100)

	users, total, err := h.store.UsersPage(r.Context(), int64((page-1)*size), int64(size))
	if err != nil {
		h.serverError(w, "lista korisnika", err)
		return
	}

	items := make([]adminUserDTO, 0, len(users))
	for i := range users {
		u := &users[i]
		items = append(items, adminUserDTO{
			ID:         u.ID.Hex(),
			FirstName:  u.FirstName,
			LastName:   u.LastName,
			Email:      u.Email,
			Role:       u.Role,
			Active:     u.Active,
			QuotaBytes: u.QuotaBytes,
			UsedBytes:  u.UsedBytes,
			CreatedAt:  u.CreatedAt,
		})
	}

	h.log.Info("administratorska aktivnost", "adminId", authFrom(r).User.ID.Hex(), "action", "list_users")
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
