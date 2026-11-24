package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type UserInfo struct {
	ID         string `json:"id"`
	QuotaBytes int64  `json:"quotaBytes"`
	UsedBytes  int64  `json:"usedBytes"`
}

func (h *Handler) GetUserInfo(
	ctx context.Context,
	authorization string,
) (*UserInfo, error) {

	if authorization == "" {
		return nil, fmt.Errorf("authorization is required")
	}

	url := strings.TrimRight(h.UsersURL, "/") + "/api/users/me"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authorization)

	response, err := h.HTTP.Do(req)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Users service returned status %d",
			response.StatusCode,
		)
	}

	var user UserInfo

	err = json.NewDecoder(response.Body).Decode(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
