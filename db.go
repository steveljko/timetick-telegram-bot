package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Entry struct {
	ID        int64
	UserID    string
	StartTime time.Time
	EndTime   sql.NullTime
	Note      string
	Active    bool
}

type Database struct {
	conn *sql.DB
}

const (
	createEntriesTableSQL = `
	CREATE TABLE IF NOT EXISTS entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL,
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP,
  note TEXT,
  active BOOLEAN NOT NULL DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  synced_at TIMESTAMP DEFAULT NULL,
	);`

	createEntrySQL           = `INSERT INTO entries (user_id, start_time, note, active) VALUES (?, ?, ?, 1)`
	getUnsyncedEntriesSQL    = `SELECT * FROM entries WHERE synced_at IS NULL`
	updateEntrySyncStatusSQL = `UPDATE entries SET synced_at = CURRENT_TIMESTAMP WHERE id = ?`
	getActiveEntrySQL        = `SELECT id, user_id, start_time, end_time, active FROM entries WHERE user_id = ? AND active = 1 LIMIT 1`
	updateEntrySQL           = `UPDATE entries SET end_time = ?, active = 0 WHERE id = ?`
	hasActiveEntrySQL        = `SELECT COUNT(*) FROM entries WHERE user_id = ? AND active = 1`
)

func NewDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database: %w", err)
	}

	db := &Database{conn: conn}
	if err := db.initDB(); err != nil {
		db.conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *Database) initDB() error {
	_, err := db.conn.Exec(createEntriesTableSQL)
	if err != nil {
		return fmt.Errorf("Failed to initialize database: %w", err)
	}
	return nil
}

// Gets list of unsynced entries
func (db *Database) GetUnsyncedEntries() ([]Entry, error) {
	entries, err := db.conn.Query(getUnsyncedEntriesSQL)
	if err != nil {
		return nil, fmt.Errorf("error querying entries: %w", err)
	}
	defer entries.Close()

	var results []Entry
	for entries.Next() {
		var entry Entry
		if err := entries.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.StartTime,
			&entry.EndTime,
			&entry.Note,
			&entry.Active,
		); err != nil {
			return nil, fmt.Errorf("error scanning entry: %w", err)
		}
		results = append(results, entry)
	}

	if err := entries.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entries: %w", err)
	}

	return results, nil
}

// Updates sync status for entry
func (db *Database) UpdateEntrySyncStatus(entryID int) error {
	_, err := db.conn.Exec(updateEntrySyncStatusSQL, entryID)
	if err != nil {
		return fmt.Errorf("error updating sync status for entry %d: %w", entryID, err)
	}
	return nil
}

// Starts entry tracking for user
func (db *Database) StartTracking(userID string, note string) error {
	// Check if user already has an active entry
	active, err := db.hasActiveEntry(userID)
	if err != nil {
		return err
	}

	if active {
		return fmt.Errorf("User already have started tracking.")
	}

	// Create new entry
	_, err = db.conn.Exec(createEntrySQL, userID, time.Now(), note)
	if err != nil {
		return fmt.Errorf("failed to create entry: %w", err)
	}

	return nil
}

// Completes active entry tracking for user
func (db *Database) StopTracking(userID string) (Entry, error) {
	// Get active entry
	entry, found, err := db.getActiveEntry(userID)
	if err != nil {
		return Entry{}, err
	}
	if !found {
		return Entry{}, fmt.Errorf("There is no active entry currently.")
	}

	// End the entry
	endTime := time.Now()
	_, err = db.conn.Exec(updateEntrySQL, endTime, entry.ID)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to end entry: %w", err)
	}

	// Update the entry object
	entry.EndTime = sql.NullTime{Time: endTime, Valid: true}
	entry.Active = false

	return entry, nil
}

// Checks if user has active (currently tracking) entry
// TODO: Make seperate log and bot messages
func (db *Database) hasActiveEntry(userID string) (bool, error) {
	var count int
	err := db.conn.QueryRow(hasActiveEntrySQL, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("Failed to check active entry: %w", err)
	}
	return count > 0, nil
}

// Get currently active tracking entry for user
// TODO: Make seperate log and bot messages
func (db *Database) getActiveEntry(userID string) (Entry, bool, error) {
	row := db.conn.QueryRow(getActiveEntrySQL, userID)

	var entry Entry
	var endTime sql.NullTime

	err := row.Scan(&entry.ID, &entry.UserID, &entry.StartTime, &endTime, &entry.Active)
	if err == sql.ErrNoRows {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("failed to get active entry: %w", err)
	}

	entry.EndTime = endTime
	return entry, true, nil
}
