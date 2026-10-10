package handlers

import (
	"net/http"
	"novel-be/internal/service"
)

// --------------------
// Health
// --------------------
func HealthCheck(sceneService service.SceneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sceneService.Ping(); err != nil {
			WriteError(w, http.StatusServiceUnavailable, "service unavailable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"status": "up"})
	}
}

// --------------------
// Root
// --------------------
func GetRoot(flow service.FlowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"message": flow.GetWelcome()})
	}
}
