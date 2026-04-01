package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "saltlick.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(4)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Conn() *sql.DB { return db.conn }
func (db *DB) Close() error  { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS flags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    enabled INTEGER DEFAULT 0,
    rollout_percent INTEGER DEFAULT 0,
    targeting_rules TEXT DEFAULT '{}',
    environment TEXT DEFAULT 'production',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_flags_name ON flags(name);

CREATE TABLE IF NOT EXISTS flag_events (
    id TEXT PRIMARY KEY,
    flag_name TEXT NOT NULL,
    user_id TEXT DEFAULT '',
    result INTEGER DEFAULT 0,
    reason TEXT DEFAULT '',
    context_json TEXT DEFAULT '{}',
    evaluated_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_events_flag ON flag_events(flag_name);
CREATE INDEX IF NOT EXISTS idx_events_time ON flag_events(evaluated_at);

CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    events TEXT DEFAULT 'flag.changed',
    enabled INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now'))
);`)
	return err
}

// --- Flag types ---

type TargetingRules struct {
	UserIDs        []string          `json:"user_ids,omitempty"`
	RolloutPercent int               `json:"rollout_percent,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
}

type Flag struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Enabled        bool           `json:"enabled"`
	RolloutPercent int            `json:"rollout_percent"`
	TargetingRules TargetingRules `json:"targeting_rules"`
	Environment    string         `json:"environment"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func (db *DB) CreateFlag(name, desc string, enabled bool, env string) (*Flag, error) {
	id := "flg_" + genID(8)
	now := time.Now().UTC().Format(time.RFC3339)
	en := 0
	if enabled {
		en = 1
	}
	if env == "" {
		env = "production"
	}
	_, err := db.conn.Exec(`INSERT INTO flags (id,name,description,enabled,environment,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?)`, id, name, desc, en, env, now, now)
	if err != nil {
		return nil, err
	}
	return &Flag{ID: id, Name: name, Description: desc, Enabled: enabled,
		Environment: env, CreatedAt: now, UpdatedAt: now, TargetingRules: TargetingRules{}}, nil
}

func (db *DB) ListFlags() ([]Flag, error) {
	rows, err := db.conn.Query(`SELECT id,name,description,enabled,rollout_percent,targeting_rules,environment,created_at,updated_at
		FROM flags ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Flag
	for rows.Next() {
		var f Flag
		var en int
		var rulesJSON string
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &en, &f.RolloutPercent, &rulesJSON, &f.Environment, &f.CreatedAt, &f.UpdatedAt); err != nil {
			continue
		}
		f.Enabled = en == 1
		json.Unmarshal([]byte(rulesJSON), &f.TargetingRules)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (db *DB) GetFlag(name string) (*Flag, error) {
	var f Flag
	var en int
	var rulesJSON string
	err := db.conn.QueryRow(`SELECT id,name,description,enabled,rollout_percent,targeting_rules,environment,created_at,updated_at
		FROM flags WHERE name=?`, name).
		Scan(&f.ID, &f.Name, &f.Description, &en, &f.RolloutPercent, &rulesJSON, &f.Environment, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	f.Enabled = en == 1
	json.Unmarshal([]byte(rulesJSON), &f.TargetingRules)
	return &f, nil
}

func (db *DB) UpdateFlag(name string, desc *string, enabled *bool, rollout *int, rules *TargetingRules, env *string) (*Flag, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if desc != nil {
		db.conn.Exec("UPDATE flags SET description=?, updated_at=? WHERE name=?", *desc, now, name)
	}
	if enabled != nil {
		en := 0
		if *enabled {
			en = 1
		}
		db.conn.Exec("UPDATE flags SET enabled=?, updated_at=? WHERE name=?", en, now, name)
	}
	if rollout != nil {
		db.conn.Exec("UPDATE flags SET rollout_percent=?, updated_at=? WHERE name=?", *rollout, now, name)
	}
	if rules != nil {
		rulesJSON, _ := json.Marshal(rules)
		db.conn.Exec("UPDATE flags SET targeting_rules=?, updated_at=? WHERE name=?", string(rulesJSON), now, name)
	}
	if env != nil {
		db.conn.Exec("UPDATE flags SET environment=?, updated_at=? WHERE name=?", *env, now, name)
	}
	return db.GetFlag(name)
}

func (db *DB) DeleteFlag(name string) error {
	db.conn.Exec("DELETE FROM flag_events WHERE flag_name=?", name)
	_, err := db.conn.Exec("DELETE FROM flags WHERE name=?", name)
	return err
}

// --- Evaluation ---

type EvalResult struct {
	Flag    string `json:"flag"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

func (db *DB) RecordEval(flagName, userID string, result bool, reason, contextJSON string) error {
	id := "ev_" + genID(10)
	en := 0
	if result {
		en = 1
	}
	_, err := db.conn.Exec(`INSERT INTO flag_events (id,flag_name,user_id,result,reason,context_json)
		VALUES (?,?,?,?,?,?)`, id, flagName, userID, en, reason, contextJSON)
	return err
}

// --- Stats ---

type FlagStats struct {
	TotalEvals  int     `json:"total_evals"`
	TrueCount   int     `json:"true_count"`
	FalseCount  int     `json:"false_count"`
	TruePercent float64 `json:"true_percent"`
}

func (db *DB) FlagStats(flagName string) (*FlagStats, error) {
	var total, trueCount int
	db.conn.QueryRow("SELECT COUNT(*) FROM flag_events WHERE flag_name=?", flagName).Scan(&total)
	db.conn.QueryRow("SELECT COUNT(*) FROM flag_events WHERE flag_name=? AND result=1", flagName).Scan(&trueCount)
	pct := 0.0
	if total > 0 {
		pct = float64(trueCount) / float64(total) * 100
	}
	return &FlagStats{
		TotalEvals:  total,
		TrueCount:   trueCount,
		FalseCount:  total - trueCount,
		TruePercent: pct,
	}, nil
}

func (db *DB) Stats() map[string]any {
	var flags, events int
	db.conn.QueryRow("SELECT COUNT(*) FROM flags").Scan(&flags)
	db.conn.QueryRow("SELECT COUNT(*) FROM flag_events").Scan(&events)
	var enabledFlags int
	db.conn.QueryRow("SELECT COUNT(*) FROM flags WHERE enabled=1").Scan(&enabledFlags)
	return map[string]any{"flags": flags, "enabled_flags": enabledFlags, "evaluations": events}
}

func (db *DB) Cleanup(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	res, err := db.conn.Exec("DELETE FROM flag_events WHERE evaluated_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DB) MonthlyEvalCount() (int, error) {
	cutoff := time.Now().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM flag_events WHERE evaluated_at >= ?", cutoff).Scan(&count)
	return count, err
}

// --- Webhooks ---

type Webhook struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Events    string `json:"events"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) ListWebhooks() ([]Webhook, error) {
	rows, err := db.conn.Query("SELECT id,url,events,enabled,created_at FROM webhooks WHERE enabled=1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Webhook
	for rows.Next() {
		var w Webhook
		var en int
		rows.Scan(&w.ID, &w.URL, &w.Events, &en, &w.CreatedAt)
		w.Enabled = en == 1
		out = append(out, w)
	}
	return out, rows.Err()
}

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
