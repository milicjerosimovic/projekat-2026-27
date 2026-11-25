package handler

import (
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"users-service/internal/model"
	"users-service/internal/store"
)

type quotaRequestBody struct {
	RequestedBytes int64 `json:"requestedBytes"`
}

type quotaRequestDTO struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	UserEmail      string     `json:"userEmail"`
	RequestedBytes int64      `json:"requestedBytes"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
}

func toQuotaDTO(q *model.QuotaRequest) quotaRequestDTO {
	dto := quotaRequestDTO{
		ID:             q.ID.Hex(),
		UserID:         q.UserID.Hex(),
		UserEmail:      q.UserEmail,
		RequestedBytes: q.RequestedBytes,
		Status:         q.Status,
		CreatedAt:      q.CreatedAt,
	}
	if !q.ProcessedAt.IsZero() {
		processed := q.ProcessedAt
		dto.ProcessedAt = &processed
	}
	return dto
}

func quotaList(list []model.QuotaRequest) []quotaRequestDTO {
	items := make([]quotaRequestDTO, 0, len(list))
	for i := range list {
		items = append(items, toQuotaDTO(&list[i]))
	}
	return items
}

func (h *Handler) createQuotaRequest(w http.ResponseWriter, r *http.Request) {
	user := authFrom(r).User

	var req quotaRequestBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan zahtev")
		return
	}
	if req.RequestedBytes <= user.QuotaBytes {
		h.rejectInput(w, r, errors.New("Trazena kvota mora biti veca od trenutne"))
		return
	}
	if req.RequestedBytes > h.cfg.MaxQuotaBytes {
		h.rejectInput(w, r, errors.New("Trazena kvota je veca od dozvoljene"))
		return
	}

	q := &model.QuotaRequest{
		UserID:         user.ID,
		UserEmail:      user.Email,
		RequestedBytes: req.RequestedBytes,
		Status:         model.QuotaPending,
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.store.CreateQuotaRequest(r.Context(), q); err != nil {
		if errors.Is(err, store.ErrPendingExists) {
			writeError(w, http.StatusConflict, "Vec imate zahtev koji ceka obradu")
			return
		}
		h.serverError(w, "upis zahteva za kvotu", err)
		return
	}

	h.log.Info("zahtev za povecanje kvote", "userId", user.ID.Hex(), "requestedBytes", req.RequestedBytes)
	writeJSON(w, http.StatusCreated, toQuotaDTO(q))
}

func (h *Handler) myQuotaRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.QuotaRequests(r.Context(), authFrom(r).User.ID, "")
	if err != nil {
		h.serverError(w, "lista zahteva za kvotu", err)
		return
	}
	writeJSON(w, http.StatusOK, quotaList(list))
}

func (h *Handler) adminQuotaRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", model.QuotaPending, model.QuotaApproved, model.QuotaRejected:
	default:
		h.rejectInput(w, r, errors.New("Nepoznat status zahteva"))
		return
	}
	list, err := h.store.QuotaRequests(r.Context(), primitive.NilObjectID, status)
	if err != nil {
		h.serverError(w, "lista zahteva za kvotu", err)
		return
	}
	writeJSON(w, http.StatusOK, quotaList(list))
}

func (h *Handler) approveQuotaRequest(w http.ResponseWriter, r *http.Request) {
	h.resolveQuotaRequest(w, r, true)
}

func (h *Handler) rejectQuotaRequest(w http.ResponseWriter, r *http.Request) {
	h.resolveQuotaRequest(w, r, false)
}

func (h *Handler) resolveQuotaRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Neispravan identifikator zahteva")
		return
	}
	admin := authFrom(r).User

	q, err := h.store.ResolveQuotaRequest(r.Context(), id, approve, admin.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Zahtev ne postoji")
		return
	}
	if errors.Is(err, store.ErrAlreadyProcessed) {
		writeError(w, http.StatusConflict, "Zahtev je vec obradjen")
		return
	}
	if err != nil {
		h.serverError(w, "obrada zahteva za kvotu", err)
		return
	}

	action := "quota_reject"
	if approve {
		action = "quota_approve"
	}
	h.log.Info("administratorska aktivnost", "adminId", admin.ID.Hex(), "action", action, "requestId", id.Hex())
	writeJSON(w, http.StatusOK, toQuotaDTO(q))
}
