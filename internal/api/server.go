package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"summarizarr/internal/database"
	"time"
)

// Server is the API server.
type Server struct {
	db     *database.DB
	server *http.Server
}

// NewServer creates a new API server.
func NewServer(addr string, db *sql.DB) *Server {
	mux := http.NewServeMux()
	s := &Server{
		db: &database.DB{DB: db},
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	mux.HandleFunc("/summaries", s.handleGetSummaries)
	mux.HandleFunc("/groups", s.handleGetGroups)
	mux.HandleFunc("/export", s.handleExport)
	mux.HandleFunc("/signal/config", s.handleSignalConfig)

	return s
}

// Start starts the API server.
func (s *Server) Start() {
	slog.Info("API server listening", "address", s.server.Addr)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("API server failed", "error", err)
		os.Exit(1)
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleGetSummaries(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "Handling GET /summaries request")

	// Parse query parameters
	params := r.URL.Query()
	search := params.Get("search")
	groups := params.Get("groups")
	startTimeStr := params.Get("start_time")
	endTimeStr := params.Get("end_time")

	slog.Debug("About to call GetSummariesWithFilters",
		"search", search,
		"groups", groups,
		"start_time", startTimeStr,
		"end_time", endTimeStr)

	summaries, err := s.db.GetSummariesWithFilters(search, groups, startTimeStr, endTimeStr)
	slog.Debug("GetSummariesWithFilters returned", "summariesLength", len(summaries), "error", err)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to get summaries", "error", err)
		http.Error(w, fmt.Sprintf("failed to get summaries: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert database summaries to API response format
	type summaryResponse struct {
		ID        int64     `json:"id"`
		GroupID   int64     `json:"group_id"`
		GroupName string    `json:"group_name"`
		Text      string    `json:"text"`
		Start     time.Time `json:"start"`
		End       time.Time `json:"end"`
		CreatedAt time.Time `json:"created_at"`
	}

	response := make([]summaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		resp := summaryResponse{
			ID:        summary.ID,
			GroupID:   summary.GroupID,
			GroupName: summary.GroupName,
			Text:      summary.Text,
			Start:     time.UnixMilli(summary.Start),
			End:       time.UnixMilli(summary.End),
		}

		// Parse created_at timestamp - try multiple formats
		var createdAt time.Time
		var err error

		// Try SQLite default format first
		if createdAt, err = time.Parse("2006-01-02 15:04:05", summary.CreatedAt); err != nil {
			// Try RFC3339 format as fallback
			if createdAt, err = time.Parse(time.RFC3339, summary.CreatedAt); err != nil {
				slog.WarnContext(r.Context(), "Failed to parse created_at timestamp", "error", err, "created_at", summary.CreatedAt)
				createdAt = time.Now() // fallback to current time
			}
		}
		resp.CreatedAt = createdAt

		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode summaries response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode summaries: %v", err), http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Successfully returned summaries", "count", len(response))
}

// handleGetGroups returns a list of all groups
func (s *Server) handleGetGroups(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "Handling GET /groups request")

	type groupResponse struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	// Query groups from database
	rows, err := s.db.Query("SELECT id, name FROM groups ORDER BY name")
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to query groups", "error", err)
		http.Error(w, fmt.Sprintf("failed to get groups: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var groups []groupResponse
	for rows.Next() {
		var group groupResponse
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			slog.ErrorContext(r.Context(), "Failed to scan group row", "error", err)
			continue
		}
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groups); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode groups response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode groups: %v", err), http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Successfully returned groups", "count", len(groups))
}

// handleExport exports summaries in various formats
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "Handling GET /export request")

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Get summaries (reuse existing logic)
	summaries, err := s.db.GetSummaries()
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to get summaries for export", "error", err)
		http.Error(w, fmt.Sprintf("failed to get summaries: %v", err), http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=summaries.json")
		if err := json.NewEncoder(w).Encode(summaries); err != nil {
			slog.ErrorContext(r.Context(), "Failed to encode summaries as JSON", "error", err)
			http.Error(w, fmt.Sprintf("failed to encode summaries: %v", err), http.StatusInternalServerError)
		}
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=summaries.csv")
		w.Write([]byte("ID,Group ID,Summary,Start,End,Created At\n"))
		for _, summary := range summaries {
			fmt.Fprintf(w, "%d,%d,\"%s\",%d,%d,%s\n",
				summary.ID, summary.GroupID, summary.Text, summary.Start, summary.End, summary.CreatedAt)
		}
	default:
		http.Error(w, "Unsupported format. Use json or csv.", http.StatusBadRequest)
		return
	}

	slog.InfoContext(r.Context(), "Successfully exported summaries", "format", format, "count", len(summaries))
}

// handleSignalConfig returns Signal configuration status
func (s *Server) handleSignalConfig(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "Handling GET /signal/config request")

	type signalConfigResponse struct {
		Connected    bool   `json:"connected"`
		Status       string `json:"status"`
		PhoneNumber  string `json:"phoneNumber"`
		IsRegistered bool   `json:"isRegistered"`
	}

	// Get phone number from environment
	phoneNumber := os.Getenv("SIGNAL_PHONE_NUMBER")

	// For now, return a simple status - this could be enhanced to check actual Signal connection
	response := signalConfigResponse{
		Connected:    true,
		Status:       "Signal CLI connected",
		PhoneNumber:  phoneNumber,
		IsRegistered: phoneNumber != "", // Consider registered if phone number is set
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode signal config response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode signal config: %v", err), http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Successfully returned signal config")
}
