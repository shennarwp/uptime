package handler

import "net/http"

// VerifyToken validates the bearer token presented by the client and is used by
// the UI to confirm a token before storing it.
// @Summary Verify API token
// @Description Checks whether the presented bearer token is valid. Returns 200 when valid, 401 otherwise.
// @Tags auth
// @Security BearerAuth
// @Success 200 "Token is valid"
// @Failure 401 "Token is invalid"
// @Router /api/auth/verify [post]
func (h *TargetHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	if !tokenMatches(r) {
		rejectUnauthorized(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}
