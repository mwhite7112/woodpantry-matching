package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mwhite7112/woodpantry-matching/internal/service"
)

func NewRouter(svc *service.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handleHealth)
	r.Get("/matches", handleGetMatches(svc))
	r.Post("/matches/query", handlePostMatchQuery(svc))

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok")) //nolint:errcheck
}

// handleGetMatches scores all recipes against the current pantry.
//
// Query params:
//   - allow_subs=true — treat substitute ingredients as equivalent when scoring
//   - max_missing=N   — include recipes missing at most N required ingredients (default 0)
func handleGetMatches(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowSubs := r.URL.Query().Get("allow_subs") == "true"

		maxMissing := 0
		if s := r.URL.Query().Get("max_missing"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 0 {
				jsonError(w, "max_missing must be a non-negative integer", http.StatusBadRequest)
				return
			}
			maxMissing = n
		}

		results, err := svc.Score(r.Context(), allowSubs, maxMissing)
		if err != nil {
			jsonError(w, "scoring failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		jsonOK(w, results)
	}
}

type matchQueryRequest struct {
	Prompt           string `json:"prompt"`
	PantryConstrained bool  `json:"pantry_constrained"`
	MaxMissing       int   `json:"max_missing"`
}

// handlePostMatchQuery is the primary "what do I cook tonight?" interface.
// Phase 1: prompt and pantry_constrained are ignored; deterministic scoring only.
func handlePostMatchQuery(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req matchQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		maxMissing := req.MaxMissing
		if maxMissing < 0 {
			maxMissing = 0
		}

		results, err := svc.Score(r.Context(), false, maxMissing)
		if err != nil {
			jsonError(w, "scoring failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		jsonOK(w, results)
	}
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}
