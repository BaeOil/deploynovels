package handlers

import (
	"net/http"
	"strconv"

	"novel-be/internal/service"
)

type DashboardHandler struct {
	service service.DashboardService
}

func NewDashboardHandler(dashboard service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: dashboard}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	summary, err := h.service.GetSummary(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get dashboard summary")
		return
	}
	WriteJSON(w, http.StatusOK, summary)
}

func (h *DashboardHandler) Trend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	months := 6
	if value := r.URL.Query().Get("months"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid months")
			return
		}
		months = parsed
	}
	timezone := r.URL.Query().Get("timezone")
	trend, err := h.service.GetTrend(r.Context(), months, timezone)
	if err != nil {
		if err.Error() == "months must be between 1 and 24" || err.Error() == "invalid timezone" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to get dashboard trend")
		return
	}
	WriteJSON(w, http.StatusOK, trend)
}
