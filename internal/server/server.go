package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stockyard-dev/stockyard-saltlick/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	port   int
	limits Limits
	client *http.Client
}

func New(db *store.DB, port int, limits Limits) *Server {
	s := &Server{
		db:     db,
		mux:    http.NewServeMux(),
		port:   port,
		limits: limits,
		client: &http.Client{Timeout: 10 * time.Second},
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Flags CRUD
	s.mux.HandleFunc("POST /api/flags", s.handleCreateFlag)
	s.mux.HandleFunc("GET /api/flags", s.handleListFlags)
	s.mux.HandleFunc("GET /api/flags/{name}", s.handleGetFlag)
	s.mux.HandleFunc("PUT /api/flags/{name}", s.handleUpdateFlag)
	s.mux.HandleFunc("DELETE /api/flags/{name}", s.handleDeleteFlag)

	// Evaluation — the hot path
	s.mux.HandleFunc("GET /api/eval/{name}", s.handleEvalFlag)
	s.mux.HandleFunc("POST /api/eval/batch", s.handleBatchEval)

	// Stats
	s.mux.HandleFunc("GET /api/flags/{name}/stats", s.handleFlagStats)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)

	// Health + UI
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ui", s.handleUI)

	// Version
	s.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"product": "stockyard-saltlick", "version": "0.1.0"})
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[saltlick] listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// --- Flag CRUD ---

func (s *Server) handleCreateFlag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
		Environment string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name is required"})
		return
	}

	// Check flag limit
	if s.limits.MaxFlags > 0 {
		flags, _ := s.db.ListFlags()
		if LimitReached(s.limits.MaxFlags, len(flags)) {
			writeJSON(w, 402, map[string]string{
				"error":   "free tier limit: " + itoa(s.limits.MaxFlags) + " flags max — upgrade to Pro",
				"upgrade": "https://stockyard.dev/saltlick/",
			})
			return
		}
	}

	// Environments require Pro
	if req.Environment != "" && req.Environment != "production" && !s.limits.Environments {
		writeJSON(w, 402, map[string]string{
			"error":   "environments require Pro — upgrade at https://stockyard.dev/saltlick/",
			"upgrade": "https://stockyard.dev/saltlick/",
		})
		return
	}

	flag, err := s.db.CreateFlag(req.Name, req.Description, req.Enabled, req.Environment)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"flag": flag})
}

func (s *Server) handleListFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := s.db.ListFlags()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if flags == nil {
		flags = []store.Flag{}
	}
	writeJSON(w, 200, map[string]any{"flags": flags, "count": len(flags)})
}

func (s *Server) handleGetFlag(w http.ResponseWriter, r *http.Request) {
	flag, err := s.db.GetFlag(r.PathValue("name"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "flag not found"})
		return
	}
	stats, _ := s.db.FlagStats(flag.Name)
	writeJSON(w, 200, map[string]any{"flag": flag, "stats": stats})
}

func (s *Server) handleUpdateFlag(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Verify flag exists
	if _, err := s.db.GetFlag(name); err != nil {
		writeJSON(w, 404, map[string]string{"error": "flag not found"})
		return
	}

	var req struct {
		Description    *string              `json:"description"`
		Enabled        *bool                `json:"enabled"`
		RolloutPercent *int                 `json:"rollout_percent"`
		TargetingRules *store.TargetingRules `json:"targeting_rules"`
		Environment    *string              `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	// Percentage rollout requires Pro
	if req.RolloutPercent != nil && *req.RolloutPercent > 0 && !s.limits.PercentRollout {
		writeJSON(w, 402, map[string]string{
			"error":   "percentage rollout requires Pro — upgrade at https://stockyard.dev/saltlick/",
			"upgrade": "https://stockyard.dev/saltlick/",
		})
		return
	}

	// User targeting requires Pro
	if req.TargetingRules != nil && len(req.TargetingRules.UserIDs) > 0 && !s.limits.UserTargeting {
		writeJSON(w, 402, map[string]string{
			"error":   "user targeting requires Pro — upgrade at https://stockyard.dev/saltlick/",
			"upgrade": "https://stockyard.dev/saltlick/",
		})
		return
	}

	// Environments require Pro
	if req.Environment != nil && *req.Environment != "production" && !s.limits.Environments {
		writeJSON(w, 402, map[string]string{
			"error":   "environments require Pro — upgrade at https://stockyard.dev/saltlick/",
			"upgrade": "https://stockyard.dev/saltlick/",
		})
		return
	}

	flag, err := s.db.UpdateFlag(name, req.Description, req.Enabled, req.RolloutPercent, req.TargetingRules, req.Environment)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	// Fire webhooks asynchronously
	if s.limits.WebhookOnChange {
		go s.fireWebhooks("flag.changed", map[string]any{"flag": flag})
	}

	writeJSON(w, 200, map[string]any{"flag": flag})
}

func (s *Server) handleDeleteFlag(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.db.GetFlag(name); err != nil {
		writeJSON(w, 404, map[string]string{"error": "flag not found"})
		return
	}
	s.db.DeleteFlag(name)
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Evaluation (the hot path) ---

func (s *Server) handleEvalFlag(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	userID := r.URL.Query().Get("user_id")
	contextParam := r.URL.Query().Get("context")

	flag, err := s.db.GetFlag(name)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "flag not found"})
		return
	}

	result, reason := s.evaluate(flag, userID, contextParam)

	// Record evaluation
	s.db.RecordEval(name, userID, result, reason, contextParam)

	writeJSON(w, 200, store.EvalResult{
		Flag:    name,
		Enabled: result,
		Reason:  reason,
	})
}

func (s *Server) handleBatchEval(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Flags   []string `json:"flags"`
		UserID  string   `json:"user_id"`
		Context string   `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	results := make([]store.EvalResult, 0, len(req.Flags))
	for _, name := range req.Flags {
		flag, err := s.db.GetFlag(name)
		if err != nil {
			results = append(results, store.EvalResult{Flag: name, Enabled: false, Reason: "not_found"})
			continue
		}
		result, reason := s.evaluate(flag, req.UserID, req.Context)
		s.db.RecordEval(name, req.UserID, result, reason, req.Context)
		results = append(results, store.EvalResult{Flag: name, Enabled: result, Reason: reason})
	}
	writeJSON(w, 200, map[string]any{"results": results, "count": len(results)})
}

// evaluate applies the flag rules and returns (enabled, reason).
func (s *Server) evaluate(flag *store.Flag, userID, contextJSON string) (bool, string) {
	// If flag is globally disabled, always false
	if !flag.Enabled {
		return false, "disabled"
	}

	rules := flag.TargetingRules

	// Check user targeting (Pro only, but rules are stored regardless)
	if len(rules.UserIDs) > 0 && userID != "" {
		for _, uid := range rules.UserIDs {
			if uid == userID {
				return true, "user_targeting"
			}
		}
	}

	// Check attribute targeting
	if len(rules.Attributes) > 0 && contextJSON != "" {
		var ctx map[string]string
		if err := json.Unmarshal([]byte(contextJSON), &ctx); err == nil {
			allMatch := true
			for k, v := range rules.Attributes {
				if ctx[k] != v {
					allMatch = false
					break
				}
			}
			if allMatch && len(rules.Attributes) > 0 {
				return true, "attribute_match"
			}
		}
	}

	// Percentage rollout
	if flag.RolloutPercent > 0 && flag.RolloutPercent < 100 {
		if userID != "" {
			// Deterministic hash-based rollout
			h := sha256.Sum256([]byte(flag.Name + ":" + userID))
			bucket := int(h[0]) % 100
			if bucket < flag.RolloutPercent {
				return true, "rollout_percent"
			}
			return false, "rollout_excluded"
		}
		// No user_id — can't do deterministic rollout, fall through to global
	}

	// If rollout is 100%, treat as fully enabled
	if flag.RolloutPercent >= 100 {
		return true, "rollout_full"
	}

	// Global on/off — flag is enabled, no targeting rules matched
	return true, "enabled"
}

// --- Webhooks ---

func (s *Server) fireWebhooks(event string, payload map[string]any) {
	hooks, err := s.db.ListWebhooks()
	if err != nil || len(hooks) == 0 {
		return
	}
	payload["event"] = event
	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	body, _ := json.Marshal(payload)

	for _, hook := range hooks {
		req, _ := http.NewRequest("POST", hook.URL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Saltlick-Event", event)
		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("[webhook] error sending to %s: %v", hook.URL, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- Stats ---

func (s *Server) handleFlagStats(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.db.GetFlag(name); err != nil {
		writeJSON(w, 404, map[string]string{"error": "flag not found"})
		return
	}
	stats, err := s.db.FlagStats(name)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, stats)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// --- Helpers ---

func itoa(n int) string { return strconv.Itoa(n) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
