package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Flag struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Rollout     int    `json:"rollout"`     // 0-100 percentage
	Environment string `json:"environment"` // all, development, staging, production
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type FlagLog struct {
	ID        string `json:"id"`
	FlagKey   string `json:"flag_key"`
	Action    string `json:"action"` // created, enabled, disabled, rollout_changed, deleted
	Detail    string `json:"detail"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "saltlick.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS flags(
		id TEXT PRIMARY KEY, key TEXT UNIQUE NOT NULL,
		name TEXT DEFAULT '', description TEXT DEFAULT '',
		enabled INTEGER DEFAULT 0, rollout INTEGER DEFAULT 100,
		environment TEXT DEFAULT 'all', tags TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now')),
		updated_at TEXT DEFAULT(datetime('now')))`)

	db.Exec(`CREATE TABLE IF NOT EXISTS flag_log(
		id TEXT PRIMARY KEY, flag_key TEXT NOT NULL,
		action TEXT NOT NULL, detail TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now')))`)

	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) log(key, action, detail string) {
	d.db.Exec(`INSERT INTO flag_log(id,flag_key,action,detail,created_at)VALUES(?,?,?,?,?)`,
		genID(), key, action, detail, now())
}

func (d *DB) CreateFlag(f *Flag) error {
	f.ID = genID()
	f.CreatedAt = now()
	f.UpdatedAt = f.CreatedAt
	if f.Environment == "" {
		f.Environment = "all"
	}
	if f.Rollout == 0 && f.Enabled {
		f.Rollout = 100
	}
	_, err := d.db.Exec(`INSERT INTO flags(id,key,name,description,enabled,rollout,environment,tags,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?,?,?)`,
		f.ID, f.Key, f.Name, f.Description, boolToInt(f.Enabled), f.Rollout, f.Environment, f.Tags, f.CreatedAt, f.UpdatedAt)
	if err == nil {
		d.log(f.Key, "created", f.Name)
	}
	return err
}

func (d *DB) GetFlag(id string) *Flag {
	var f Flag
	var enabled int
	if d.db.QueryRow(`SELECT id,key,name,description,enabled,rollout,environment,tags,created_at,updated_at FROM flags WHERE id=?`, id).
		Scan(&f.ID, &f.Key, &f.Name, &f.Description, &enabled, &f.Rollout, &f.Environment, &f.Tags, &f.CreatedAt, &f.UpdatedAt) != nil {
		return nil
	}
	f.Enabled = enabled != 0
	return &f
}

func (d *DB) GetByKey(key string) *Flag {
	var f Flag
	var enabled int
	if d.db.QueryRow(`SELECT id,key,name,description,enabled,rollout,environment,tags,created_at,updated_at FROM flags WHERE key=?`, key).
		Scan(&f.ID, &f.Key, &f.Name, &f.Description, &enabled, &f.Rollout, &f.Environment, &f.Tags, &f.CreatedAt, &f.UpdatedAt) != nil {
		return nil
	}
	f.Enabled = enabled != 0
	return &f
}

func (d *DB) ListFlags() []Flag {
	rows, _ := d.db.Query(`SELECT id,key,name,description,enabled,rollout,environment,tags,created_at,updated_at FROM flags ORDER BY created_at DESC`)
	if rows == nil {
		return []Flag{}
	}
	defer rows.Close()
	var out []Flag
	for rows.Next() {
		var f Flag
		var enabled int
		rows.Scan(&f.ID, &f.Key, &f.Name, &f.Description, &enabled, &f.Rollout, &f.Environment, &f.Tags, &f.CreatedAt, &f.UpdatedAt)
		f.Enabled = enabled != 0
		out = append(out, f)
	}
	if out == nil {
		return []Flag{}
	}
	return out
}

func (d *DB) UpdateFlag(f *Flag) error {
	f.UpdatedAt = now()
	_, err := d.db.Exec(`UPDATE flags SET name=?,description=?,enabled=?,rollout=?,environment=?,tags=?,updated_at=? WHERE id=?`,
		f.Name, f.Description, boolToInt(f.Enabled), f.Rollout, f.Environment, f.Tags, f.UpdatedAt, f.ID)
	return err
}

func (d *DB) ToggleFlag(id string, enabled bool) error {
	f := d.GetFlag(id)
	if f == nil {
		return fmt.Errorf("not found")
	}
	_, err := d.db.Exec(`UPDATE flags SET enabled=?,updated_at=? WHERE id=?`, boolToInt(enabled), now(), id)
	if err == nil {
		action := "disabled"
		if enabled {
			action = "enabled"
		}
		d.log(f.Key, action, f.Name)
	}
	return err
}

func (d *DB) SetRollout(id string, pct int) error {
	f := d.GetFlag(id)
	if f == nil {
		return fmt.Errorf("not found")
	}
	_, err := d.db.Exec(`UPDATE flags SET rollout=?,updated_at=? WHERE id=?`, pct, now(), id)
	if err == nil {
		d.log(f.Key, "rollout_changed", fmt.Sprintf("%d%%", pct))
	}
	return err
}

func (d *DB) DeleteFlag(id string) error {
	f := d.GetFlag(id)
	if f != nil {
		d.log(f.Key, "deleted", f.Name)
	}
	_, err := d.db.Exec(`DELETE FROM flags WHERE id=?`, id)
	return err
}

// Evaluate returns whether a flag is active for a given context
func (d *DB) Evaluate(key string, userID string) map[string]any {
	f := d.GetByKey(key)
	if f == nil {
		return map[string]any{"key": key, "enabled": false, "reason": "flag_not_found"}
	}
	if !f.Enabled {
		return map[string]any{"key": key, "enabled": false, "reason": "flag_disabled"}
	}
	if f.Rollout >= 100 {
		return map[string]any{"key": key, "enabled": true, "reason": "fully_rolled_out"}
	}
	// Simple hash-based rollout
	hash := 0
	for _, c := range userID + key {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	inRollout := (hash % 100) < f.Rollout
	reason := "rollout_included"
	if !inRollout {
		reason = "rollout_excluded"
	}
	return map[string]any{"key": key, "enabled": inRollout, "rollout": f.Rollout, "reason": reason}
}

func (d *DB) ListLog(limit int) []FlagLog {
	if limit <= 0 {
		limit = 50
	}
	rows, _ := d.db.Query(`SELECT id,flag_key,action,detail,created_at FROM flag_log ORDER BY created_at DESC LIMIT ?`, limit)
	if rows == nil {
		return []FlagLog{}
	}
	defer rows.Close()
	var out []FlagLog
	for rows.Next() {
		var l FlagLog
		rows.Scan(&l.ID, &l.FlagKey, &l.Action, &l.Detail, &l.CreatedAt)
		out = append(out, l)
	}
	if out == nil {
		return []FlagLog{}
	}
	return out
}

func (d *DB) Stats() map[string]any {
	var total, enabled, disabled int
	d.db.QueryRow(`SELECT COUNT(*) FROM flags`).Scan(&total)
	d.db.QueryRow(`SELECT COUNT(*) FROM flags WHERE enabled=1`).Scan(&enabled)
	disabled = total - enabled
	return map[string]any{"total": total, "enabled": enabled, "disabled": disabled}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
