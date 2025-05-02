package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type APIHandler struct {
	app *App
	db  *Database
}

func NewAPIHandler(app *App, db *Database) *APIHandler {
	return &APIHandler{
		app: app,
		db:  db,
	}
}

func SetupRoutes(app *App, db *Database) http.Handler {
	handler := NewAPIHandler(app, db)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/entries", handler.getUnimportedEntries)
	mux.HandleFunc("POST /api/entries/mark", handler.markEntriesAsImported)

	return mux
}

func StartAPIServer(app *App, db *Database, port int) error {
	handler := SetupRoutes(app, db)
	addr := fmt.Sprintf(":%d", port)

	log.Printf("Starting API server on %s\n", addr)
	return http.ListenAndServe(addr, handler)
}

type ErrorCode string

const (
	// General error codes
	MISSING_PARAMS  ErrorCode = "MISSING_PARAMS"
	INVALID_REQUEST ErrorCode = "INVALID_REQUEST"
	INTERNAL_ERROR  ErrorCode = "INTERNAL_ERROR"

	// Entry related error codes
	NO_ENTRIES           ErrorCode = "NO_ENTRIES"
	NO_ENTRIES_TO_IMPORT ErrorCode = "NO_ENTRIES_TO_IMPORT"
	FAILED_FETCH         ErrorCode = "FAILED_FETCH"
	ENTRY_NOT_FOUND      ErrorCode = "ENTRY_NOT_FOUND"
	IMPORT_FAILED        ErrorCode = "IMPORT_FAILED"
)

type Response struct {
	Success bool      `json:"success"`
	Code    ErrorCode `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
	Data    any       `json:"data,omitempty"`
}

// Sends JSON error response with the specified status code and message
func RespondWithError(w http.ResponseWriter, statusCode int, code ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Code:    code,
		Message: message,
	})
}

// Sends successful JSON response with the specified data
func RespondWithJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

// Sends JSON response with a message and specified success status
func RespondWithMessage(w http.ResponseWriter, statusCode int, message string, success bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: message,
	})
}

func (h *APIHandler) getUnimportedEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := h.db.GetUnimportedEntries()
	if err != nil {
		log.Println(fmt.Sprintf("Failed to retrieve entries: %v", err))
		RespondWithError(w, http.StatusInternalServerError, FAILED_FETCH, "Failed to fetch unimported entries.")
		return
	}

	if len(entries) == 0 {
		RespondWithError(w, http.StatusNotFound, NO_ENTRIES_TO_IMPORT, "There are no entries for importing.")
		return
	}

	RespondWithJSON(w, http.StatusOK, struct {
		Total   int     `json:"total"`
		Entries []Entry `json:"entries"`
	}{
		Total:   len(entries),
		Entries: entries,
	})
}

func (h *APIHandler) markEntriesAsImported(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntryIDs []int64 `json:"entry_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.EntryIDs) == 0 {
		log.Println("User tried to submit request without 'entry_ids' parameter.")
		RespondWithError(w, http.StatusBadRequest, MISSING_PARAMS, "You must provide 'entry_ids' into body.")
		return
	}

	// get unimported count
	entries, err := h.db.GetUnimportedEntries()
	if err != nil {
		log.Println(fmt.Sprintf("Failed to retrieve unimported entries: %v", err))
		return
	}
	remainingUnimportedCount := len(entries)

	if remainingUnimportedCount == 0 {
		log.Println("There are no entries to import.")
		RespondWithError(w, http.StatusNotFound, NO_ENTRIES_TO_IMPORT, "There are no entries to import.")
		return
	}

	// Update import status for each entry
	var importedCount int
	for _, entryID := range req.EntryIDs {
		exists, isUnimported, err := h.db.CheckEntry(int(entryID))
		if err != nil {
			log.Println(fmt.Sprintf("Error checking entry %d import status: %v", entryID, err))
			continue
		}

		// Skips if entry doesn't exist or is already imported
		if !exists {
			log.Println(fmt.Sprintf("Entry %d not found!", entryID))
			continue
		}

		// Skips if entry with this id is already imported
		if !isUnimported {
			log.Println(fmt.Sprintf("Entry %d is already imported, skipping.", entryID))
			continue
		}

		if err := h.db.UpdateEntryImportStatus(int(entryID)); err != nil {
			log.Printf("Failed to mark entry %d as imported: %v", entryID, err)
			continue
		}
		importedCount++
	}

	RespondWithJSON(w, http.StatusOK, struct {
		ImportedCount  int `json:"imported_count"`
		RemainingCount int `json:"remaining_count"`
	}{
		ImportedCount:  importedCount,
		RemainingCount: remainingUnimportedCount,
	})
}
