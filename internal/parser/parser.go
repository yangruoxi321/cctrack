package parser

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ksred/cctrack/internal/calculator"
	"github.com/ksred/cctrack/internal/store"
)

type Parser struct {
	store *store.Store
}

func New(s *store.Store) *Parser {
	return &Parser{store: s}
}

// ParseAll discovers and parses all JSONL files in the log directory.
// Returns the number of files parsed and sessions affected.
func (p *Parser) ParseAll(logDir string) (int, int, error) {
	files, err := DiscoverFiles(logDir)
	if err != nil {
		return 0, 0, fmt.Errorf("discovering files: %w", err)
	}

	filesParsed := 0
	sessionsAffected := make(map[string]bool)

	for _, path := range files {
		sessions, err := p.ParseFile(path)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			continue
		}
		if len(sessions) > 0 {
			filesParsed++
			for _, s := range sessions {
				sessionsAffected[s] = true
			}
		}
	}

	return filesParsed, len(sessionsAffected), nil
}

// ParseFile reads new data from a single JSONL file (from last known offset).
// Returns the session IDs that were affected.
func (p *Parser) ParseFile(path string) ([]string, error) {
	offset, err := p.store.GetFileOffset(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Check for truncation
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if offset > fi.Size() {
		// File was truncated/rotated. Roll back the contributions this file
		// previously made to the sessions table before re-parsing from the
		// start; otherwise the additive UpsertSession would double-count.
		if err := p.rollbackFileContributions(path); err != nil {
			return nil, fmt.Errorf("rolling back contributions for %s: %w", path, err)
		}
		offset = 0
	}
	if offset == fi.Size() {
		return nil, nil // nothing new
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	info := ExtractSessionInfo(path)

	// Read all events, deduplicating by requestId (last event wins)
	type eventEntry struct {
		event RawEvent
		order int
	}
	byRequestID := make(map[string]eventEntry)
	var noRequestID []RawEvent
	orderCounter := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event RawEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue // skip malformed lines
		}

		// Only process assistant messages with usage data
		if event.Type != "assistant" {
			continue
		}
		if event.Message.Usage.OutputTokens == 0 && event.Message.Usage.InputTokens == 0 &&
			event.Message.Usage.CacheCreationInputTokens == 0 && event.Message.Usage.CacheReadInputTokens == 0 {
			continue
		}

		if event.RequestID != "" {
			byRequestID[event.RequestID] = eventEntry{event: event, order: orderCounter}
		} else {
			noRequestID = append(noRequestID, event)
		}
		orderCounter++
	}

	// Aggregate token usage per session
	type sessionAgg struct {
		model      string
		slug       string
		sessionID  string
		timestamp  string
		input      int64
		output     int64
		cacheRead  int64
		cacheWrite int64
	}
	sessions := make(map[string]*sessionAgg)

	// Collect request-level records for timeline feature
	var requestRecords []store.RequestRecord

	processEvent := func(event RawEvent, requestID string) {
		sid := event.SessionID
		if sid == "" {
			sid = info.SessionID
		}
		if sid == "" {
			return
		}

		agg, ok := sessions[sid]
		if !ok {
			agg = &sessionAgg{sessionID: sid}
			sessions[sid] = agg
		}

		if event.Message.Model != "" {
			agg.model = event.Message.Model
		}
		if event.Slug != "" {
			agg.slug = event.Slug
		}
		if event.Timestamp > agg.timestamp {
			agg.timestamp = event.Timestamp
		}

		u := event.Message.Usage
		agg.input += u.InputTokens
		agg.output += u.OutputTokens
		agg.cacheRead += u.CacheReadInputTokens
		agg.cacheWrite += u.CacheCreationInputTokens

		// Store per-request record if we have a requestID
		if requestID != "" {
			usage := calculator.TokenUsage{
				InputTokens:      u.InputTokens,
				OutputTokens:     u.OutputTokens,
				CacheReadTokens:  u.CacheReadInputTokens,
				CacheWriteTokens: u.CacheCreationInputTokens,
			}
			cost := calculator.Calculate(event.Message.Model, usage)
			requestRecords = append(requestRecords, store.RequestRecord{
				RequestID:        requestID,
				SessionID:        sid,
				Timestamp:        event.Timestamp,
				Model:            event.Message.Model,
				InputTokens:      u.InputTokens,
				OutputTokens:     u.OutputTokens,
				CacheReadTokens:  u.CacheReadInputTokens,
				CacheWriteTokens: u.CacheCreationInputTokens,
				Cost:             cost.TotalCost,
			})
		}
	}

	for reqID, entry := range byRequestID {
		processEvent(entry.event, reqID)
	}
	for _, event := range noRequestID {
		processEvent(event, "")
	}

	// Pre-compute session deltas (with cost) so we can write them in a single
	// transaction below.
	type pendingSession struct {
		delta store.SessionDelta
		cost  float64
	}
	pending := make(map[string]pendingSession, len(sessions))
	for sid, agg := range sessions {
		usage := calculator.TokenUsage{
			InputTokens:      agg.input,
			OutputTokens:     agg.output,
			CacheReadTokens:  agg.cacheRead,
			CacheWriteTokens: agg.cacheWrite,
		}
		cost := calculator.Calculate(agg.model, usage)

		pending[sid] = pendingSession{
			delta: store.SessionDelta{
				ID:              sid,
				Project:         info.Project,
				Slug:            agg.slug,
				Model:           agg.model,
				Timestamp:       agg.timestamp,
				DeltaInput:      agg.input,
				DeltaOutput:     agg.output,
				DeltaCacheRead:  agg.cacheRead,
				DeltaCacheWrite: agg.cacheWrite,
				DeltaCost:       cost.TotalCost,
			},
			cost: cost.TotalCost,
		}
	}

	// Single transaction for the entire write phase: session upserts, request
	// upserts, per-file contribution updates, and the offset bump.
	var affectedIDs []string
	newOffset := fi.Size()
	err = p.store.WithTx(func(tx *sql.Tx) error {
		for sid, ps := range pending {
			if err := p.store.UpsertSessionTx(tx, ps.delta); err != nil {
				return fmt.Errorf("upsert session %s: %w", sid, err)
			}
			if err := p.store.AddFileContributionTx(tx, store.FileContribution{
				FilePath:   path,
				SessionID:  sid,
				Input:      ps.delta.DeltaInput,
				Output:     ps.delta.DeltaOutput,
				CacheRead:  ps.delta.DeltaCacheRead,
				CacheWrite: ps.delta.DeltaCacheWrite,
				Cost:       ps.cost,
			}); err != nil {
				return fmt.Errorf("record contribution %s: %w", sid, err)
			}
			affectedIDs = append(affectedIDs, sid)
		}

		for _, rec := range requestRecords {
			if err := p.store.UpsertRequestTx(tx, rec); err != nil {
				return fmt.Errorf("upsert request %s: %w", rec.RequestID, err)
			}
		}

		if err := p.store.SetFileOffsetTx(tx, path, newOffset); err != nil {
			return fmt.Errorf("updating offset: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return affectedIDs, nil
}

// rollbackFileContributions subtracts everything this file previously
// contributed to the sessions table and then clears the contribution rows
// for the file. Used when truncation/rotation is detected so a re-parse
// from offset 0 doesn't double-count.
func (p *Parser) rollbackFileContributions(path string) error {
	contribs, err := p.store.GetFileContributions(path)
	if err != nil {
		return err
	}
	if len(contribs) == 0 {
		// Nothing recorded (e.g. file was tracked before this feature shipped).
		// Best we can do is drop any stale rows and proceed.
		return p.store.ClearFileContributions(path)
	}
	return p.store.WithTx(func(tx *sql.Tx) error {
		for _, c := range contribs {
			if err := p.store.SubtractFromSessionTx(tx, c.SessionID,
				c.Input, c.Output, c.CacheRead, c.CacheWrite, c.Cost); err != nil {
				return fmt.Errorf("subtract from session %s: %w", c.SessionID, err)
			}
		}
		return p.store.ClearFileContributionsTx(tx, path)
	})
}

// ParseFiles parses a specific set of files (used by watcher for incremental updates).
func (p *Parser) ParseFiles(paths []string) ([]string, error) {
	var allSessions []string
	for _, path := range paths {
		sessions, err := p.ParseFile(path)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			continue
		}
		allSessions = append(allSessions, sessions...)
	}
	return allSessions, nil
}
