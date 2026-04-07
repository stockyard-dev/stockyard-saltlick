package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-saltlick/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}

	s.mux.HandleFunc("GET /api/flags", s.listFlags)
	s.mux.HandleFunc("POST /api/flags", s.createFlag)
	s.mux.HandleFunc("GET /api/flags/{id}", s.getFlag)
	s.mux.HandleFunc("PUT /api/flags/{id}", s.updateFlag)
	s.mux.HandleFunc("PATCH /api/flags/{id}/toggle", s.toggleFlag)
	s.mux.HandleFunc("PATCH /api/flags/{id}/rollout", s.setRollout)
	s.mux.HandleFunc("DELETE /api/flags/{id}", s.deleteFlag)
	s.mux.HandleFunc("GET /api/evaluate/{key}", s.evaluate)
	s.mux.HandleFunc("GET /api/log", s.listLog)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/saltlick/"})
	})
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	json.NewEncoder(w).Encode(v)
}
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

func (s *Server) listFlags(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"flags": s.db.ListFlags()})
}

func (s *Server) createFlag(w http.ResponseWriter, r *http.Request) {
	if s.limits.MaxItems > 0 && len(s.db.ListFlags()) >= s.limits.MaxItems {
		we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/saltlick/")
		return
	}
	var f store.Flag
	json.NewDecoder(r.Body).Decode(&f)
	if f.Key == "" {
		we(w, 400, "key required")
		return
	}
	if err := s.db.CreateFlag(&f); err != nil {
		we(w, 400, err.Error())
		return
	}
	wj(w, 201, s.db.GetByKey(f.Key))
}

func (s *Server) getFlag(w http.ResponseWriter, r *http.Request) {
	f := s.db.GetFlag(r.PathValue("id"))
	if f == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, f)
}

func (s *Server) updateFlag(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetFlag(r.PathValue("id"))
	if existing == nil {
		we(w, 404, "not found")
		return
	}
	var patch store.Flag
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID
	patch.Key = existing.Key
	patch.CreatedAt = existing.CreatedAt
	if patch.Name == "" {
		patch.Name = existing.Name
	}
	s.db.UpdateFlag(&patch)
	wj(w, 200, s.db.GetFlag(patch.ID))
}

func (s *Server) toggleFlag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := s.db.ToggleFlag(r.PathValue("id"), body.Enabled); err != nil {
		we(w, 404, err.Error())
		return
	}
	wj(w, 200, s.db.GetFlag(r.PathValue("id")))
}

func (s *Server) setRollout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rollout int `json:"rollout"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Rollout < 0 || body.Rollout > 100 {
		we(w, 400, "rollout must be 0-100")
		return
	}
	if err := s.db.SetRollout(r.PathValue("id"), body.Rollout); err != nil {
		we(w, 404, err.Error())
		return
	}
	wj(w, 200, s.db.GetFlag(r.PathValue("id")))
}

func (s *Server) deleteFlag(w http.ResponseWriter, r *http.Request) {
	if s.db.GetFlag(r.PathValue("id")) == nil {
		we(w, 404, "not found")
		return
	}
	s.db.DeleteFlag(r.PathValue("id"))
	wj(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) evaluate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	userID := r.URL.Query().Get("user")
	if userID == "" {
		userID = "anonymous"
	}
	wj(w, 200, s.db.Evaluate(key, userID))
}

func (s *Server) listLog(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"log": s.db.ListLog(50)})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) { wj(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	wj(w, 200, map[string]any{"service": "saltlick", "status": "ok", "flags": stats["total"], "enabled": stats["enabled"]})
}

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "saltlick", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "saltlick", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}
