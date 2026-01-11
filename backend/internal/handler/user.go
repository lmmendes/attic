package handler

import (
	"net/http"

	"github.com/mendelui/attic/internal/auth"
)

type CurrentUserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"display_name,omitempty"`
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	response := CurrentUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}

	writeJSON(w, http.StatusOK, response)
}
