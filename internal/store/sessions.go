package store

import (
	"database/sql"
	"fmt"
)

type Session struct {
	ID              string  `json:"id"`
	Project         string  `json:"project"`
	Slug            string  `json:"slug"`
	Model           string  `json:"model"`
	StartedAt       string  `json:"started_at"`
	LastActivity    string  `json:"last_activity"`
	TotalInput      int64   `json:"total_input"`
	TotalOutput     int64   `json:"total_output"`
	TotalCacheRead  int64   `json:"total_cache_read"`
	TotalCacheWrite int64   `json:"total_cache_write"`
	TotalCost       float64 `json:"total_cost"`
}

func (s *Session) TotalTokens() int64 {
	return s.TotalInput + s.TotalOutput + s.TotalCacheRead + s.TotalCacheWrite
}

type SessionDelta struct {
	ID              string
	Project         string
	Slug            string
	Model           string
	Timestamp       string
	DeltaInput      int64
	DeltaOutput     int64
	DeltaCacheRead  int64
	DeltaCacheWrite int64
	DeltaCost       float64
}

// upsertSessionExec runs the additive session upsert against either *sql.DB
// or *sql.Tx (any value satisfying the small execer interface).
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

const upsertSessionSQL = `
		INSERT INTO sessions (id, project, slug, model, started_at, last_activity,
			total_input, total_output, total_cache_read, total_cache_write, total_cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			slug          = CASE WHEN excluded.slug != '' THEN excluded.slug ELSE sessions.slug END,
			model         = CASE WHEN excluded.model != '' THEN excluded.model ELSE sessions.model END,
			last_activity = CASE WHEN excluded.last_activity > sessions.last_activity THEN excluded.last_activity ELSE sessions.last_activity END,
			total_input   = sessions.total_input   + excluded.total_input,
			total_output  = sessions.total_output  + excluded.total_output,
			total_cache_read  = sessions.total_cache_read  + excluded.total_cache_read,
			total_cache_write = sessions.total_cache_write + excluded.total_cache_write,
			total_cost    = sessions.total_cost    + excluded.total_cost
	`

func upsertSessionExec(e execer, d SessionDelta) error {
	_, err := e.Exec(upsertSessionSQL,
		d.ID, d.Project, d.Slug, d.Model, d.Timestamp, d.Timestamp,
		d.DeltaInput, d.DeltaOutput, d.DeltaCacheRead, d.DeltaCacheWrite, d.DeltaCost)
	return err
}

// UpsertSession adds token deltas to an existing session or creates a new one.
// Token counts are ADDITIVE — new values add to existing totals.
func (s *Store) UpsertSession(d SessionDelta) error {
	return upsertSessionExec(s.db, d)
}

// UpsertSessionTx is the transaction-aware variant of UpsertSession.
func (s *Store) UpsertSessionTx(tx *sql.Tx, d SessionDelta) error {
	return upsertSessionExec(tx, d)
}

// SubtractFromSessionTx decrements a session's token totals and cost by the
// supplied values. Used when re-parsing a truncated file: we first roll back
// its previous contribution, then re-add the new totals.
func (s *Store) SubtractFromSessionTx(tx *sql.Tx, sessionID string, input, output, cacheRead, cacheWrite int64, cost float64) error {
	_, err := tx.Exec(`
		UPDATE sessions SET
			total_input       = MAX(0, total_input       - ?),
			total_output      = MAX(0, total_output      - ?),
			total_cache_read  = MAX(0, total_cache_read  - ?),
			total_cache_write = MAX(0, total_cache_write - ?),
			total_cost        = MAX(0, total_cost        - ?)
		WHERE id = ?`,
		input, output, cacheRead, cacheWrite, cost, sessionID)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
		FROM sessions WHERE id = ?`, id)
	sess := &Session{}
	err := row.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
		&sess.StartedAt, &sess.LastActivity,
		&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
		&sess.TotalCost)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// --- Request-level tracking ---

type RequestRecord struct {
	RequestID        string  `json:"request_id"`
	SessionID        string  `json:"session_id"`
	Timestamp        string  `json:"timestamp"`
	Model            string  `json:"model"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	Cost             float64 `json:"cost"`
}

const upsertRequestSQL = `
		INSERT INTO requests (request_id, session_id, timestamp, model,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(request_id) DO UPDATE SET
			timestamp = excluded.timestamp,
			model = excluded.model,
			input_tokens = excluded.input_tokens,
			output_tokens = excluded.output_tokens,
			cache_read_tokens = excluded.cache_read_tokens,
			cache_write_tokens = excluded.cache_write_tokens,
			cost = excluded.cost
	`

func upsertRequestExec(e execer, r RequestRecord) error {
	_, err := e.Exec(upsertRequestSQL,
		r.RequestID, r.SessionID, r.Timestamp, r.Model,
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.Cost)
	return err
}

func (s *Store) UpsertRequest(r RequestRecord) error {
	return upsertRequestExec(s.db, r)
}

// UpsertRequestTx is the transaction-aware variant of UpsertRequest.
func (s *Store) UpsertRequestTx(tx *sql.Tx, r RequestRecord) error {
	return upsertRequestExec(tx, r)
}

func (s *Store) GetSessionRequests(sessionID string) ([]RequestRecord, error) {
	rows, err := s.db.Query(`
		SELECT request_id, session_id, timestamp, model,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost
		FROM requests WHERE session_id = ?
		ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []RequestRecord
	for rows.Next() {
		var r RequestRecord
		if err := rows.Scan(&r.RequestID, &r.SessionID, &r.Timestamp, &r.Model,
			&r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheWriteTokens,
			&r.Cost); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	return recs, nil
}

var allowedSortColumns = map[string]string{
	"cost":    "total_cost",
	"date":    "last_activity",
	"started": "started_at",
	"tokens":  "(total_input + total_output + total_cache_read + total_cache_write)",
	"model":   "model",
	"project": "project",
}

func (s *Store) ListSessions(limit, offset int, sortBy, sortDir string) ([]Session, int, error) {
	col, ok := allowedSortColumns[sortBy]
	if !ok {
		col = "total_cost"
	}
	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}

	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT id, project, slug, model, started_at, last_activity,
		total_input, total_output, total_cache_read, total_cache_write, total_cost
		FROM sessions ORDER BY %s %s LIMIT ? OFFSET ?`, col, dir)

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Slug, &sess.Model,
			&sess.StartedAt, &sess.LastActivity,
			&sess.TotalInput, &sess.TotalOutput, &sess.TotalCacheRead, &sess.TotalCacheWrite,
			&sess.TotalCost); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, total, nil
}

// --- Per-file contribution tracking ---
//
// file_session_contributions records what each (file_path, session_id) pair
// has contributed to the aggregate sessions row. This lets us roll back a
// file's contribution when the file is truncated (rotated/replaced) so the
// re-parse doesn't double-count.

type FileContribution struct {
	FilePath   string
	SessionID  string
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
	Cost       float64
}

// GetFileContributions returns all per-session contributions previously
// recorded for the given file path.
func (s *Store) GetFileContributions(filePath string) ([]FileContribution, error) {
	rows, err := s.db.Query(`
		SELECT file_path, session_id, input, output, cache_read, cache_write, cost
		FROM file_session_contributions
		WHERE file_path = ?`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FileContribution
	for rows.Next() {
		var c FileContribution
		if err := rows.Scan(&c.FilePath, &c.SessionID, &c.Input, &c.Output,
			&c.CacheRead, &c.CacheWrite, &c.Cost); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ClearFileContributions removes all per-session contribution rows for the
// given file path. Used after we've rolled back the file's contribution from
// the sessions table on truncation.
func (s *Store) ClearFileContributions(filePath string) error {
	_, err := s.db.Exec(`DELETE FROM file_session_contributions WHERE file_path = ?`, filePath)
	return err
}

// ClearFileContributionsTx is the tx-aware variant.
func (s *Store) ClearFileContributionsTx(tx *sql.Tx, filePath string) error {
	_, err := tx.Exec(`DELETE FROM file_session_contributions WHERE file_path = ?`, filePath)
	return err
}

// AddFileContributionTx adds the given delta to the per-file contribution row
// for (filePath, sessionID), creating it if missing. This is additive — the
// caller passes the delta that was just observed in this parse pass.
//
// Callers should normally invoke this from inside the same transaction that
// applied the matching UpsertSession delta, so the two stay in sync.
func (s *Store) AddFileContributionTx(tx *sql.Tx, c FileContribution) error {
	_, err := tx.Exec(`
		INSERT INTO file_session_contributions
			(file_path, session_id, input, output, cache_read, cache_write, cost)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path, session_id) DO UPDATE SET
			input       = file_session_contributions.input       + excluded.input,
			output      = file_session_contributions.output      + excluded.output,
			cache_read  = file_session_contributions.cache_read  + excluded.cache_read,
			cache_write = file_session_contributions.cache_write + excluded.cache_write,
			cost        = file_session_contributions.cost        + excluded.cost`,
		c.FilePath, c.SessionID, c.Input, c.Output, c.CacheRead, c.CacheWrite, c.Cost)
	return err
}

// RecordFileContribution upserts (additive) a per-file/session contribution
// outside of an existing transaction. Convenience wrapper for callers that
// don't need to batch with other writes.
func (s *Store) RecordFileContribution(c FileContribution) error {
	return s.WithTx(func(tx *sql.Tx) error {
		return s.AddFileContributionTx(tx, c)
	})
}
