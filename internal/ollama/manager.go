package ollama

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Manager handles Ollama binary management and process lifecycle
type Manager struct {
	binPath    string
	modelsPath string
	host       string
	process    *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager creates a new Ollama manager
func NewManager(modelsPath, host string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Convert to absolute path
	absModelsPath, err := filepath.Abs(modelsPath)
	if err != nil {
		absModelsPath = modelsPath // fallback to relative path
	}

	// Determine binary path
	binPath := filepath.Join(absModelsPath, "ollama")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	return &Manager{
		binPath:    binPath,
		modelsPath: absModelsPath,
		host:       host,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// EnsureInstalled downloads and installs Ollama binary if not present
func (m *Manager) EnsureInstalled(ctx context.Context) error {
	// Check if binary already exists
	if _, err := os.Stat(m.binPath); err == nil {
		slog.InfoContext(ctx, "Ollama binary already exists", "path", m.binPath)
		return nil
	}

	slog.InfoContext(ctx, "Downloading Ollama binary...")

	// Create models directory
	if err := os.MkdirAll(m.modelsPath, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Determine download URL based on OS and architecture
	downloadURL, err := m.getDownloadURL()
	if err != nil {
		return fmt.Errorf("failed to determine download URL: %w", err)
	}

	slog.InfoContext(ctx, "Downloading from", "url", downloadURL)

	// Download with progress tracking
	if err := m.downloadWithProgress(ctx, downloadURL); err != nil {
		return fmt.Errorf("failed to download Ollama: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(m.binPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	slog.InfoContext(ctx, "Ollama binary downloaded successfully", "path", m.binPath)
	return nil
}

// getDownloadURL returns the appropriate download URL for the current platform
func (m *Manager) getDownloadURL() (string, error) {
	baseURL := "https://ollama.com/download/"

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return baseURL + "ollama-linux-amd64.tgz", nil
		case "arm64":
			return baseURL + "ollama-linux-arm64.tgz", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return baseURL + "ollama-darwin.tgz", nil
		case "arm64":
			return baseURL + "ollama-darwin-arm64.tgz", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "windows":
		return baseURL + "ollama-windows-amd64.zip", nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// downloadWithProgress downloads and extracts Ollama with progress tracking
func (m *Manager) downloadWithProgress(ctx context.Context, url string) error {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Download file
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create progress reader
	progressReader := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		ctx:    ctx,
	}

	// Extract directly to models directory
	if filepath.Ext(url) == ".tgz" {
		return m.extractTarGz(progressReader)
	}
	// For now, we only support .tgz files
	return fmt.Errorf("unsupported archive format")
}

// extractTarGz extracts a tar.gz archive to the models directory
func (m *Manager) extractTarGz(reader io.Reader) error {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// We only want the ollama binary
		if filepath.Base(header.Name) == "ollama" || filepath.Base(header.Name) == "ollama.exe" {
			file, err := os.Create(m.binPath)
			if err != nil {
				return fmt.Errorf("failed to create binary file: %w", err)
			}

			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}

			// Make the binary executable
			if err := os.Chmod(m.binPath, 0755); err != nil {
				return fmt.Errorf("failed to make binary executable: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("ollama binary not found in archive")
}

// Start starts the Ollama server as a detached process
func (m *Manager) Start(ctx context.Context) error {
	if m.process != nil {
		return fmt.Errorf("ollama process already running")
	}

	// Check if binary exists and is executable
	if stat, err := os.Stat(m.binPath); err != nil {
		return fmt.Errorf("ollama binary not found at %s: %w", m.binPath, err)
	} else {
		slog.DebugContext(ctx, "Ollama binary found", "path", m.binPath, "size", stat.Size(), "mode", stat.Mode())
	}

	slog.InfoContext(ctx, "Starting Ollama server", "host", m.host, "models_path", m.modelsPath)

	// Create command with environment variables
	m.process = exec.CommandContext(m.ctx, m.binPath, "serve")
	m.process.Env = append(os.Environ(),
		fmt.Sprintf("OLLAMA_HOST=%s", m.host),
		fmt.Sprintf("OLLAMA_MODELS=%s", m.modelsPath),
	)

	// Set working directory
	m.process.Dir = m.modelsPath

	// Start the process
	if err := m.process.Start(); err != nil {
		return fmt.Errorf("failed to start ollama process: %w", err)
	}

	slog.InfoContext(ctx, "Ollama server started", "pid", m.process.Process.Pid)

	// Wait for server to be ready
	if err := m.waitForReady(ctx); err != nil {
		m.Stop()
		return fmt.Errorf("ollama server failed to start: %w", err)
	}

	return nil
}

// waitForReady waits for Ollama server to be ready to accept requests
func (m *Manager) waitForReady(ctx context.Context) error {
	url := fmt.Sprintf("http://%s/api/tags", m.host)

	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Try to connect
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				continue
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				slog.InfoContext(ctx, "Ollama server is ready")
				return nil
			}
		}
	}

	return fmt.Errorf("ollama server did not become ready within 30 seconds")
}

// Stop stops the Ollama server
func (m *Manager) Stop() error {
	if m.process == nil {
		return nil
	}

	slog.Info("Stopping Ollama server")

	// Cancel context to signal shutdown
	m.cancel()

	// Try graceful shutdown first
	if m.process.Process != nil {
		m.process.Process.Signal(os.Interrupt)

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- m.process.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown takes too long
			m.process.Process.Kill()
			<-done
		case <-done:
			// Graceful shutdown completed
		}
	}

	m.process = nil
	slog.Info("Ollama server stopped")
	return nil
}

// IsRunning checks if the Ollama process is still running
func (m *Manager) IsRunning() bool {
	return m.process != nil && m.process.Process != nil
}

// progressReader wraps an io.Reader to provide download progress
type progressReader struct {
	reader  io.Reader
	total   int64
	read    int64
	ctx     context.Context
	lastLog time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	// Log progress every second
	if time.Since(pr.lastLog) > time.Second {
		if pr.total > 0 {
			progress := float64(pr.read) / float64(pr.total) * 100
			slog.InfoContext(pr.ctx, "Download progress", "percent", fmt.Sprintf("%.1f%%", progress),
				"downloaded_mb", pr.read/1024/1024, "total_mb", pr.total/1024/1024)
		} else {
			slog.InfoContext(pr.ctx, "Download progress", "downloaded_mb", pr.read/1024/1024)
		}
		pr.lastLog = time.Now()
	}

	return n, err
}
