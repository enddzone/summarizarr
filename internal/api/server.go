package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Server is the API server.
type Server struct {
	db     *sql.DB
	server *http.Server
}

// NewServer creates a new API server.
func NewServer(addr string, db *sql.DB) *Server {
	mux := http.NewServeMux()
	s := &Server{
		db: db,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	mux.HandleFunc("/summaries", s.handleGetSummaries)

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
	rows, err := s.db.QueryContext(r.Context(), "SELECT summary_text, start_timestamp, end_timestamp, created_at FROM summaries ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query summaries: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type summary struct {
		Text      string    `json:"text"`
		Start     time.Time `json:"start"`
		End       time.Time `json:"end"`
		CreatedAt time.Time `json:"created_at"`
	}

	var summaries []summary
	for rows.Next() {
		var s summary
		var start, end int64
		if err := rows.Scan(&s.Text, &start, &end, &s.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("failed to scan summary: %v", err), http.StatusInternalServerError)
			return
		}
		s.Start = time.Unix(start, 0)
		s.End = time.Unix(end, 0)
		summaries = append(summaries, s)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode summaries: %v", err), http.StatusInternalServerError)
	}
}