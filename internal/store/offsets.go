package store

import (
	"database/sql"
	"errors"
	"time"
)

func (s *Store) GetFileOffset(path string) (int64, error) {
	var offset int64
	err := s.db.QueryRow("SELECT offset FROM file_offsets WHERE path = ?", path).Scan(&offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil // not found = start from beginning
		}
		return 0, err
	}
	return offset, nil
}

const setFileOffsetSQL = `
		INSERT INTO file_offsets (path, offset, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET offset = excluded.offset, updated_at = excluded.updated_at
	`

func (s *Store) SetFileOffset(path string, offset int64) error {
	_, err := s.db.Exec(setFileOffsetSQL, path, offset, time.Now().UTC().Format(time.RFC3339))
	return err
}

// SetFileOffsetTx is the transaction-aware variant of SetFileOffset.
func (s *Store) SetFileOffsetTx(tx *sql.Tx, path string, offset int64) error {
	_, err := tx.Exec(setFileOffsetSQL, path, offset, time.Now().UTC().Format(time.RFC3339))
	return err
}
