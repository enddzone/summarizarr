package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"summarizarr/internal/database"
	"summarizarr/internal/version"
	"time"
)

// Server is the API server.
type Server struct {
	db     *database.DB
	server *http.Server
}

// ServerOptions holds configuration options for the server
type ServerOptions struct {
	SignalURL      string
	ValidateSignal bool
}

// ServerOption is a functional option for configuring the server
type ServerOption func(*ServerOptions)

// WithSignalURL sets a custom Signal CLI URL
func WithSignalURL(url string) ServerOption {
	return func(opts *ServerOptions) {
		opts.SignalURL = url
	}
}

// WithSignalValidation enables or disables Signal CLI validation on startup
func WithSignalValidation(validate bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.ValidateSignal = validate
	}
}

// NewServer creates a new API server with default options.
// Maintained for backward compatibility.
func NewServer(addr string, db *sql.DB, frontendFS fs.FS) *Server {
	return NewServerWithOptions(addr, db, frontendFS)
}

// NewServerWithOptions creates a new API server with configurable options.
func NewServerWithOptions(addr string, db *sql.DB, frontendFS fs.FS, options ...ServerOption) *Server {
	// Apply default options
	opts := &ServerOptions{
		SignalURL:      getSignalURL(),
		ValidateSignal: true,
	}
	
	// Apply provided options
	for _, option := range options {
		option(opts)
	}
	
	slog.Info("Initializing API server", "signal_url", opts.SignalURL, "listen_addr", addr)
	
	// Validate Signal URL if enabled
	if opts.ValidateSignal {
		if err := validateSignalURL(opts.SignalURL); err != nil {
			slog.Warn("Signal CLI validation failed during startup", "signal_url", opts.SignalURL, "error", err)
		} else {
			slog.Info("Signal CLI connectivity verified", "signal_url", opts.SignalURL)
		}
	}

	mux := http.NewServeMux()
	s := &Server{
		db: &database.DB{DB: db},
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	// API routes
	mux.HandleFunc("/api/summaries", s.handleGetSummaries)
	mux.HandleFunc("/api/summaries/", s.handleDeleteSummary) // DELETE /api/summaries/{id}
	mux.HandleFunc("/api/groups", s.handleGetGroups)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/signal/config", s.handleSignalConfig)
	mux.HandleFunc("/api/signal/register", s.handleSignalRegister)
	mux.HandleFunc("/api/signal/status", s.handleSignalStatus)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/health", s.handleHealth)

	// Frontend static files
	if frontendFS != nil {
		mux.Handle("/", s.serveFrontend(frontendFS))
	}

	return s
}

// handleDeleteSummary handles DELETE /api/summaries/{id}
func (s *Server) handleDeleteSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	// Expected path: /api/summaries/{id}
	// Trim prefix and extract the id
	idStr := r.URL.Path[len("/api/summaries/"):]
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	var id int64
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteSummary(id); err != nil {
		slog.ErrorContext(r.Context(), "Failed to delete summary", "error", err, "id", id)
		http.Error(w, "failed to delete summary", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write response", "error", err)
	}
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
	sort := params.Get("sort")

	slog.Debug("About to call GetSummariesWithFilters",
		"search", search,
		"groups", groups,
		"start_time", startTimeStr,
		"end_time", endTimeStr,
		"sort", sort)

	summaries, err := s.db.GetSummariesWithFilters(search, groups, startTimeStr, endTimeStr, sort)
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
		if _, err := w.Write([]byte("ID,Group ID,Summary,Start,End,Created At\n")); err != nil {
			slog.ErrorContext(r.Context(), "Failed to write CSV header", "error", err)
			return
		}
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

	// Check if the phone number is actually registered with Signal CLI
	isRegistered := false
	status := "Signal CLI not configured"
	connected := false

	if phoneNumber != "" {
		isRegistered = s.checkSignalRegistration(phoneNumber)
		if isRegistered {
			status = "Signal CLI registered and ready"
			connected = true
		} else {
			status = "Signal CLI connected but phone number not registered"
			connected = true
		}
	}

	response := signalConfigResponse{
		Connected:    connected,
		Status:       status,
		PhoneNumber:  phoneNumber,
		IsRegistered: isRegistered,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode signal config response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode signal config: %v", err), http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Successfully returned signal config", "isRegistered", isRegistered)
}

// checkSignalRegistration checks if a phone number is registered with Signal CLI
func (s *Server) checkSignalRegistration(phoneNumber string) bool {
	signalURL := getSignalURL()

	// Try to get the list of registered accounts from Signal CLI
	url := fmt.Sprintf("http://%s/v1/accounts", signalURL)
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Failed to check Signal CLI accounts", "error", err, "url", url)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Signal CLI accounts check returned non-200 status", "status", resp.StatusCode, "url", url)
		return false
	}

	var accounts []string
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		slog.Error("Failed to decode Signal CLI accounts response", "error", err)
		return false
	}

	// Check if our phone number is in the list of registered accounts
	for _, account := range accounts {
		if account == phoneNumber {
			slog.Info("Phone number found in registered accounts", "phoneNumber", phoneNumber)
			return true
		}
	}

	slog.Info("Phone number not found in registered accounts", "phoneNumber", phoneNumber, "registeredAccounts", accounts)
	return false
}

// handleSignalRegister handles registration flow for Signal
func (s *Server) handleSignalRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	slog.InfoContext(r.Context(), "Handling POST /signal/register request")

	type registerRequest struct {
		PhoneNumber string `json:"phoneNumber"`
	}

	type registerResponse struct {
		QrCodeUrl    string `json:"qrCodeUrl,omitempty"`
		IsRegistered bool   `json:"isRegistered"`
		Message      string `json:"message"`
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(r.Context(), "Failed to decode register request", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.PhoneNumber == "" {
		http.Error(w, "phoneNumber is required", http.StatusBadRequest)
		return
	}

	// Get Signal CLI URL
	signalURL := getSignalURL()

	// Check if already registered first
	if s.checkSignalRegistration(req.PhoneNumber) {
		response := registerResponse{
			IsRegistered: true,
			Message:      "Phone number is already registered",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate QR code for device linking (recommended approach)
	qrURL := fmt.Sprintf("http://%s/v1/qrcodelink?device_name=summarizarr", signalURL)

	response := registerResponse{
		QrCodeUrl:    qrURL,
		IsRegistered: false,
		Message:      "Scan QR code with Signal to link device",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode register response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Generated QR code for Signal registration", "phoneNumber", req.PhoneNumber)
}

// handleSignalStatus checks Signal registration status
func (s *Server) handleSignalStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	slog.InfoContext(r.Context(), "Handling GET /signal/status request")

	type statusResponse struct {
		IsRegistered bool   `json:"isRegistered"`
		PhoneNumber  string `json:"phoneNumber"`
		Message      string `json:"message"`
	}

	phoneNumber := os.Getenv("SIGNAL_PHONE_NUMBER")
	isRegistered := false
	message := "Phone number not configured"

	if phoneNumber != "" {
		isRegistered = s.checkSignalRegistration(phoneNumber)
		if isRegistered {
			message = "Phone number is registered and ready"
		} else {
			message = "Phone number not registered with Signal CLI"
		}
	}

	response := statusResponse{
		IsRegistered: isRegistered,
		PhoneNumber:  phoneNumber,
		Message:      message,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode status response", "error", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "Successfully returned Signal status", "isRegistered", isRegistered)
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	versionInfo := version.Get()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(versionInfo); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode version response", "error", err)
		http.Error(w, "failed to encode version response", http.StatusInternalServerError)
		return
	}
}

// handleHealth returns health status for container health checks
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "summarizarr",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode health response", "error", err)
		http.Error(w, "failed to encode health response", http.StatusInternalServerError)
		return
	}
}

// serveFrontend creates a handler for serving frontend static files
// getSignalURL returns the Signal CLI URL from environment variable or default
func getSignalURL() string {
	signalURL := strings.TrimSpace(os.Getenv("SIGNAL_URL"))
	if signalURL == "" {
		return "localhost:8080"
	}
	
	// Remove any protocol prefix for consistency
	signalURL = strings.TrimPrefix(signalURL, "http://")
	signalURL = strings.TrimPrefix(signalURL, "https://")
	
	return signalURL
}

// validateSignalURL validates that the Signal URL is reachable
func validateSignalURL(signalURL string) error {
	if signalURL == "" {
		return fmt.Errorf("signal URL is empty")
	}
	
	// Basic URL validation - ensure it contains a valid host:port pattern
	if !strings.Contains(signalURL, ":") {
		return fmt.Errorf("signal URL must include port (e.g., localhost:8080)")
	}
	
	// Test connectivity to the Signal CLI service
	url := fmt.Sprintf("http://%s/v1/health", signalURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("signal CLI not reachable at %s: %w", signalURL, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("signal CLI health check failed at %s: status %d", signalURL, resp.StatusCode)
	}
	
	return nil
}

// validateAndCleanPath validates and cleans a URL path to prevent directory traversal attacks
func validateAndCleanPath(urlPath string) (string, error) {
	// Handle root path specially
	if urlPath == "/" {
		return "", nil // Empty path for root, will be handled by SPA routing
	}
	
	// Remove leading slash
	cleanPath := strings.TrimPrefix(urlPath, "/")
	
	// Check for directory traversal patterns BEFORE cleaning
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: directory traversal attempt")
	}
	
	// Use filepath.Clean for proper OS path handling
	cleanPath = filepath.Clean(cleanPath)
	
	// Convert back to forward slashes for consistency (filepath.Clean may use OS separator)
	cleanPath = filepath.ToSlash(cleanPath)
	
	// Handle single dot (current directory) - this can result from filepath.Clean on traversal attempts
	if cleanPath == "." {
		return "", nil // Empty path for SPA routing
	}
	
	// Comprehensive directory traversal checks
	if cleanPath == ".." || 
		strings.HasPrefix(cleanPath, "../") || 
		strings.Contains(cleanPath, "/../") ||
		strings.HasSuffix(cleanPath, "/..") ||
		strings.Contains(cleanPath, "\\") ||
		strings.Contains(cleanPath, "\x00") {
		return "", fmt.Errorf("invalid path: directory traversal attempt")
	}
	
	// Ensure path doesn't start with a dot (hidden files), but allow empty paths
	if cleanPath != "" && strings.HasPrefix(cleanPath, ".") {
		return "", fmt.Errorf("invalid path: access to hidden files not allowed")
	}
	
	// File extension whitelist for security (only for non-empty paths)
	if cleanPath != "" && !isAllowedFileExtension(cleanPath) {
		return "", fmt.Errorf("invalid path: file type not allowed")
	}
	
	return cleanPath, nil
}

// isAllowedFileExtension checks if the file extension is in the whitelist
func isAllowedFileExtension(path string) bool {
	// Allow files without extensions (for SPA routing)
	if !strings.Contains(path, ".") {
		return true
	}
	
	// Allowed extensions for frontend assets
	allowedExts := []string{
		".html", ".htm",
		".css",
		".js", ".mjs",
		".json",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp",
		".woff", ".woff2", ".ttf", ".eot",
		".txt", ".xml",
	}
	
	// Blocked extensions that should never be served
	blockedExts := []string{
		".php", ".asp", ".aspx", ".jsp",
		".sh", ".bat", ".cmd", ".exe", ".com",
		".env", ".config", ".ini", ".conf",
		".key", ".pem", ".crt", ".cer",
		".log", ".bak", ".backup", ".tmp", ".temp",
		".sql", ".db", ".sqlite", ".sqlite3",
	}
	
	lowerPath := strings.ToLower(path)
	
	// Check for blocked extensions anywhere in the filename
	for _, ext := range blockedExts {
		if strings.Contains(lowerPath, ext) {
			return false
		}
	}
	
	// Check if final extension is in allowed list
	for _, ext := range allowedExts {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}
	
	return false
}

func (s *Server) serveFrontend(frontendFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}

		// Sanitize and validate the path to prevent directory traversal
		cleanPath, err := validateAndCleanPath(urlPath)
		if err != nil {
			slog.WarnContext(r.Context(), "Invalid path request blocked", "path", urlPath, "error", err)
			http.NotFound(w, r)
			return
		}

		// Try to serve the exact file first
		content, err := fs.ReadFile(frontendFS, cleanPath)
		if err != nil {
			// If file not found, serve index.html for SPA routing
			content, err = fs.ReadFile(frontendFS, "index.html")
			if err != nil {
				slog.ErrorContext(r.Context(), "Failed to read index.html", "error", err)
				http.Error(w, "page not found", http.StatusNotFound)
				return
			}
			// Set correct content type for HTML
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			// Set appropriate content type based on file extension
			if strings.HasSuffix(urlPath, ".css") {
				w.Header().Set("Content-Type", "text/css")
			} else if strings.HasSuffix(urlPath, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			} else if strings.HasSuffix(urlPath, ".html") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			} else if strings.HasSuffix(urlPath, ".json") {
				w.Header().Set("Content-Type", "application/json")
			} else if strings.HasSuffix(urlPath, ".png") {
				w.Header().Set("Content-Type", "image/png")
			} else if strings.HasSuffix(urlPath, ".svg") {
				w.Header().Set("Content-Type", "image/svg+xml")
			} else if strings.HasSuffix(urlPath, ".ico") {
				w.Header().Set("Content-Type", "image/x-icon")
			}
		}

		w.Write(content)
	})
}
