package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Entry struct {
	ID         int64        `json:"id"`
	UserID     string       `json:"user_id"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    sql.NullTime `json:"end_time"`
	Note       string       `json:"note"`
	Active     bool         `json:"active"`
	ImportedAt sql.NullTime `json:"imported_at"`
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
  imported_at TIMESTAMP DEFAULT NULL
	)`

	createEntrySQL             = `INSERT INTO entries (user_id, start_time, note, active) VALUES (?, ?, ?, 1)`
	getUnimportedEntriesSQL    = `SELECT id, user_id, start_time, end_time, note, active, imported_at FROM entries WHERE imported_at IS NULL`
	updateEntryImportStatusSQL = `UPDATE entries SET imported_at = CURRENT_TIMESTAMP WHERE id = ?`
	checkEntrySQL              = `SELECT COUNT(*), CASE WHEN imported_at IS NULL THEN 1 ELSE 0 END FROM entries WHERE id = ?`
	getActiveEntrySQL          = `SELECT id, user_id, start_time, end_time, active FROM entries WHERE user_id = ? AND active = 1 LIMIT 1`
	updateEntrySQL             = `UPDATE entries SET end_time = ?, active = 0 WHERE id = ?`
	hasActiveEntrySQL          = `SELECT COUNT(*) FROM entries WHERE user_id = ? AND active = 1`
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

// Gets list of unimported entries
func (db *Database) GetUnimportedEntries() ([]Entry, error) {
	entries, err := db.conn.Query(getUnimportedEntriesSQL)
	if err != nil {
		return nil, fmt.Errorf("Error querying entries: %w", err)
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
			&entry.ImportedAt,
		); err != nil {
			return nil, fmt.Errorf("Error scanning entry: %w", err)
		}
		results = append(results, entry)
	}

	if err := entries.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating entries: %w", err)
	}

	return results, nil
}

// Updates imported status for entry
func (db *Database) UpdateEntryImportStatus(entryID int) error {
	_, err := db.conn.Exec(updateEntryImportStatusSQL, entryID)
	if err != nil {
		return fmt.Errorf("Error updating import status for entry %d: %w", entryID, err)
	}
	return nil
}

// Checks if entry exists and their import status
func (db *Database) CheckEntry(entryID int) (exists bool, isImported bool, err error) {
	var count, unimported int
	err = db.conn.QueryRow(checkEntrySQL, entryID).Scan(&count, &unimported)
	if err != nil {
		return false, false, fmt.Errorf("Error checking entry %d: %w", entryID, err)
	}

	return count > 0, unimported == 1, nil
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
		return Entry{}, fmt.Errorf("Failed to end entry: %w", err)
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
		return Entry{}, false, fmt.Errorf("Failed to get active entry: %w", err)
	}

	entry.EndTime = endTime
	return entry, true, nil
}
