package api

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"summarizarr/internal/database"
	"summarizarr/internal/version"
	"time"
)

// CacheConfig defines caching configuration for different file types
type CacheConfig struct {
	MaxAge         int  // Cache duration in seconds
	MustRevalidate bool // Force revalidation
	NoCache        bool // Disable caching
	UseETag        bool // Enable ETag generation
}

// Security constants for QR code proxy
const (
	maxDeviceNameLength = 50
	maxResponseSize     = 5 * 1024 * 1024 // 5MB limit for response body
	defaultQRTimeout    = 30 * time.Second
	maxRetries          = 3
	retryDelay          = 1 * time.Second
)

// Device name validation regex - only alphanumeric, hyphens, underscores
var deviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
	mux.HandleFunc("/api/signal/qrcode", s.handleSignalQrCode)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/health", s.handleHealth)

	// Frontend static files
	if frontendFS != nil {
		mux.Handle("/", s.serveFrontend(frontendFS))
	}

	return s
}

// getCacheConfig returns cache configuration based on file extension
func getCacheConfig(path string) CacheConfig {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".html", ".htm":
		// HTML files: no-cache, must-revalidate for fresh content
		return CacheConfig{
			MaxAge:         0,
			MustRevalidate: true,
			NoCache:        true,
			UseETag:        true,
		}
	case ".css", ".js", ".mjs":
		// CSS/JS files: long-term caching with ETag for cache busting
		return CacheConfig{
			MaxAge:         31536000, // 1 year
			MustRevalidate: false,
			NoCache:        false,
			UseETag:        true,
		}
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp":
		// Images: medium-term caching
		return CacheConfig{
			MaxAge:         2592000, // 30 days
			MustRevalidate: false,
			NoCache:        false,
			UseETag:        true,
		}
	case ".json":
		// JSON files: short-term caching
		return CacheConfig{
			MaxAge:         300, // 5 minutes
			MustRevalidate: true,
			NoCache:        false,
			UseETag:        true,
		}
	case ".woff", ".woff2", ".ttf", ".eot":
		// Fonts: long-term caching
		return CacheConfig{
			MaxAge:         31536000, // 1 year
			MustRevalidate: false,
			NoCache:        false,
			UseETag:        true,
		}
	default:
		// Default: short-term caching with revalidation
		return CacheConfig{
			MaxAge:         300, // 5 minutes
			MustRevalidate: true,
			NoCache:        false,
			UseETag:        true,
		}
	}
}

// setResponseCacheHeaders sets appropriate cache headers based on configuration
func setResponseCacheHeaders(w http.ResponseWriter, config CacheConfig, content []byte) {
	if config.NoCache {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else {
		cacheControl := fmt.Sprintf("max-age=%d", config.MaxAge)
		if config.MustRevalidate {
			cacheControl += ", must-revalidate"
		}
		w.Header().Set("Cache-Control", cacheControl)
	}

	// Generate and set ETag if enabled
	if config.UseETag && len(content) > 0 {
		hash := md5.Sum(content)
		etag := `"` + hex.EncodeToString(hash[:]) + `"`
		w.Header().Set("ETag", etag)
	}

	// Set Last-Modified to current time for all resources
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
}

// getContentType returns the appropriate MIME type for a file path
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Use mime package for standard detection first
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		return contentType
	}

	// Custom mappings for common web assets
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// checkConditionalRequest checks if the request can be served from cache
func checkConditionalRequest(r *http.Request, etag string) bool {
	// Check If-None-Match header
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		// Simple comparison - in production, this should handle multiple ETags
		return ifNoneMatch == etag
	}

	return false
}

// setAPIResponseHeaders sets appropriate headers for API responses
func setAPIResponseHeaders(w http.ResponseWriter, r *http.Request, data []byte, cacheSeconds int) bool {
	// Generate ETag for the response data
	hash := md5.Sum(data)
	etag := `"` + hex.EncodeToString(hash[:]) + `"`

	// Check conditional request
	if checkConditionalRequest(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return true // Indicates that response was served from cache
	}

	// Set cache headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	if cacheSeconds > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", cacheSeconds))
	} else {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	}

	return false // Indicates that full response should be sent
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

	// Encode response to JSON first to generate ETag
	responseData, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode summaries response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode summaries: %v", err), http.StatusInternalServerError)
		return
	}

	// Set cache headers (5 minutes cache for summaries)
	if served := setAPIResponseHeaders(w, r, responseData, 300); served {
		return // Response served from cache (304 Not Modified)
	}

	// Write the response
	if _, err := w.Write(responseData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write summaries response", "error", err)
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
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			slog.ErrorContext(r.Context(), "Failed to close groups rows", "error", cerr)
		}
	}()

	groups := []groupResponse{}
	for rows.Next() {
		var group groupResponse
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			slog.ErrorContext(r.Context(), "Failed to scan group row", "error", err)
			continue
		}
		groups = append(groups, group)
	}

	// Encode response to JSON first to generate ETag
	responseData, err := json.Marshal(groups)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode groups response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode groups: %v", err), http.StatusInternalServerError)
		return
	}

	// Set cache headers (10 minutes cache for groups as they change infrequently)
	if served := setAPIResponseHeaders(w, r, responseData, 600); served {
		return // Response served from cache (304 Not Modified)
	}

	// Write the response
	if _, err := w.Write(responseData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write groups response", "error", err)
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
			if _, err := fmt.Fprintf(w, "%d,%d,\"%s\",%d,%d,%s\n",
				summary.ID, summary.GroupID, summary.Text, summary.Start, summary.End, summary.CreatedAt); err != nil {
				slog.ErrorContext(r.Context(), "Failed to write CSV row", "error", err)
				return
			}
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

	// Encode response to JSON first to generate ETag
	responseData, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode signal config response", "error", err)
		http.Error(w, fmt.Sprintf("failed to encode signal config: %v", err), http.StatusInternalServerError)
		return
	}

	// Set cache headers (30 seconds cache for config as it may change during setup)
	if served := setAPIResponseHeaders(w, r, responseData, 30); served {
		return // Response served from cache (304 Not Modified)
	}

	// Write the response
	if _, err := w.Write(responseData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write signal config response", "error", err)
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Error("Failed to close accounts response body", "error", cerr)
		}
	}()

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

	// Check if already registered first
	if s.checkSignalRegistration(req.PhoneNumber) {
		response := registerResponse{
			IsRegistered: true,
			Message:      "Phone number is already registered",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.ErrorContext(r.Context(), "Failed to encode already-registered response", "error", err)
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	// Generate QR code for device linking (recommended approach)
	// Use our proxy endpoint instead of direct Signal CLI URL for frontend compatibility
	qrURL := "/api/signal/qrcode?device_name=summarizarr"

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

// handleSignalQrCode proxies QR code requests to Signal CLI with security enhancements
func (s *Server) handleSignalQrCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	slog.InfoContext(r.Context(), "Handling GET /signal/qrcode request")

	// Get Signal CLI URL
	signalURL := getSignalURL()

	// Get and validate device name from query parameters
	deviceName := r.URL.Query().Get("device_name")
	if deviceName == "" {
		deviceName = "summarizarr"
	}

	// Validate device name for security
	if err := validateDeviceName(deviceName); err != nil {
		slog.WarnContext(r.Context(), "Invalid device name provided", "device_name", deviceName, "error", err)
		http.Error(w, fmt.Sprintf("invalid device name: %v", err), http.StatusBadRequest)
		return
	}

	// URL-encode the device name to prevent injection
	encodedDeviceName := url.QueryEscape(deviceName)

	// Build the Signal CLI QR code URL with proper encoding
	qrCodeURL := fmt.Sprintf("http://%s/v1/qrcodelink?device_name=%s", signalURL, encodedDeviceName)

	// Attempt the request with retry logic
	var resp *http.Response
	var err error
	timeout := getQRTimeout()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a new request to Signal CLI
		proxyReq, reqErr := http.NewRequestWithContext(r.Context(), "GET", qrCodeURL, nil)
		if reqErr != nil {
			slog.ErrorContext(r.Context(), "Failed to create proxy request", "error", reqErr, "attempt", attempt)
			if attempt == maxRetries {
				http.Error(w, "failed to create proxy request", http.StatusInternalServerError)
				return
			}
			time.Sleep(retryDelay)
			continue
		}

		// Forward the request to Signal CLI with configurable timeout
		client := &http.Client{Timeout: timeout}
		resp, err = client.Do(proxyReq)
		if err != nil {
			slog.WarnContext(r.Context(), "Failed to proxy QR code request", "error", err, "url", qrCodeURL, "attempt", attempt)
			if attempt == maxRetries {
				http.Error(w, "failed to fetch QR code after retries", http.StatusBadGateway)
				return
			}
			time.Sleep(retryDelay)
			continue
		}

		// Success - break out of retry loop
		break
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.ErrorContext(r.Context(), "Failed to close QR code response body", "error", cerr)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(r.Context(), "Signal CLI returned non-200 status for QR code", "status", resp.StatusCode, "url", qrCodeURL)
		http.Error(w, "QR code generation failed", resp.StatusCode)
		return
	}

	// Copy only safe headers from Signal CLI response
	for name, values := range resp.Header {
		if isSafeHeader(name) {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body with size limit to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	bytesWritten, err := io.Copy(w, limitedReader)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to copy QR code response", "error", err)
		return
	}

	// Log if we hit the size limit
	if bytesWritten == maxResponseSize {
		slog.WarnContext(r.Context(), "QR code response truncated due to size limit", "bytes_written", bytesWritten, "limit", maxResponseSize)
	}

	slog.InfoContext(r.Context(), "Successfully proxied QR code request",
		"device_name", deviceName,
		"bytes_written", bytesWritten,
		"attempts", func() int {
			for i := 1; i <= maxRetries; i++ {
				if err == nil {
					return i
				}
			}
			return maxRetries
		}())
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	versionInfo := version.Get()

	// Encode response to JSON first to generate ETag
	responseData, err := json.Marshal(versionInfo)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode version response", "error", err)
		http.Error(w, "failed to encode version response", http.StatusInternalServerError)
		return
	}

	// Set cache headers (1 hour cache for version info as it changes infrequently)
	if served := setAPIResponseHeaders(w, r, responseData, 3600); served {
		return // Response served from cache (304 Not Modified)
	}

	// Write the response
	if _, err := w.Write(responseData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write version response", "error", err)
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

	// Encode response to JSON first
	responseData, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode health response", "error", err)
		http.Error(w, "failed to encode health response", http.StatusInternalServerError)
		return
	}

	// No cache for health endpoint (always return fresh status)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write the response
	if _, err := w.Write(responseData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write health response", "error", err)
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

// validateDeviceName validates the device name parameter for QR code generation
func validateDeviceName(deviceName string) error {
	if deviceName == "" {
		return fmt.Errorf("device name cannot be empty")
	}

	if len(deviceName) > maxDeviceNameLength {
		return fmt.Errorf("device name too long (max %d characters)", maxDeviceNameLength)
	}

	if !deviceNameRegex.MatchString(deviceName) {
		return fmt.Errorf("device name can only contain alphanumeric characters, hyphens, and underscores")
	}

	return nil
}

// getQRTimeout returns the configured timeout for QR code requests
func getQRTimeout() time.Duration {
	if timeoutStr := os.Getenv("QR_TIMEOUT_SECONDS"); timeoutStr != "" {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultQRTimeout
}

// isSafeHeader checks if a header should be copied from the upstream response
func isSafeHeader(headerName string) bool {
	// Convert to lowercase for case-insensitive comparison
	name := strings.ToLower(headerName)

	// Headers safe to copy
	safeHeaders := map[string]bool{
		"content-type":     true,
		"content-length":   true,
		"cache-control":    true,
		"expires":          true,
		"last-modified":    true,
		"etag":             true,
		"content-encoding": true,
	}

	// Block potentially sensitive headers
	blockedHeaders := map[string]bool{
		"authorization":   true,
		"cookie":          true,
		"set-cookie":      true,
		"x-forwarded-for": true,
		"x-real-ip":       true,
		"x-api-key":       true,
		"server":          true,
	}

	if blockedHeaders[name] {
		return false
	}

	return safeHeaders[name]
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Error("Failed to close health response body", "error", cerr)
		}
	}()

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
		originalPath := urlPath
		isSPARoute := false

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
			// Mark this as SPA routing
			isSPARoute = true
			cleanPath = "index.html"
		}

		// Get cache configuration for the file type
		var cacheConfig CacheConfig
		if isSPARoute {
			// Use HTML cache config for SPA routes
			cacheConfig = getCacheConfig("index.html")
		} else {
			cacheConfig = getCacheConfig(cleanPath)
		}

		// Generate ETag if enabled
		var etag string
		if cacheConfig.UseETag && len(content) > 0 {
			hash := md5.Sum(content)
			etag = `"` + hex.EncodeToString(hash[:]) + `"`
		}

		// Check conditional request (If-None-Match)
		if etag != "" && checkConditionalRequest(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Set content type using enhanced detection
		var contentType string
		if isSPARoute {
			contentType = "text/html; charset=utf-8"
		} else {
			contentType = getContentType(cleanPath)
		}
		w.Header().Set("Content-Type", contentType)

		// Set cache headers
		setResponseCacheHeaders(w, cacheConfig, content)

		// Log cache strategy for debugging
		slog.DebugContext(r.Context(), "Serving static asset",
			"path", originalPath,
			"file", cleanPath,
			"spa_route", isSPARoute,
			"content_type", contentType,
			"cache_max_age", cacheConfig.MaxAge,
			"has_etag", etag != "",
		)

		if _, err := w.Write(content); err != nil {
			slog.ErrorContext(r.Context(), "Failed to write response content", "error", err, "path", originalPath)
		}
	})
}
