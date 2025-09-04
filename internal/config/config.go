package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	LogLevel              slog.Level
	ListenAddr            string
	PhoneNumber           string
	SignalURL             string
	DatabasePath          string
	LocalModel            string
	OllamaKeepAlive       string
	OllamaHost            string
	SummarizationInterval string

	// Generic provider configuration
	AIProvider string

	// OpenAI configuration
	OpenAIAPIKey  string
	OpenAIModel   string
	OpenAIBaseURL string

	// Groq configuration
	GroqAPIKey  string
	GroqModel   string
	GroqBaseURL string

	// Gemini configuration
	GeminiAPIKey  string
	GeminiModel   string
	GeminiBaseURL string

	// Claude configuration
	ClaudeAPIKey  string
	ClaudeModel   string
	ClaudeBaseURL string
}

// New creates a new Config from environment variables.
func New() *Config {
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "summarizarr.db" // default path
	}

	localModel := os.Getenv("LOCAL_MODEL")
	if localModel == "" {
		localModel = "llama3.2:1b" // default model - smaller memory footprint
	}

	ollamaKeepAlive := os.Getenv("OLLAMA_KEEP_ALIVE")
	if ollamaKeepAlive == "" {
		ollamaKeepAlive = "5m" // default
	}

	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "127.0.0.1:11434" // default
	}

	summarizationInterval := os.Getenv("SUMMARIZATION_INTERVAL")
	if summarizationInterval == "" {
		summarizationInterval = "12h" // default
	}

	// rotation feature removed

	signalURL := os.Getenv("SIGNAL_URL")
	if signalURL == "" {
		signalURL = "signal-cli-rest-api:8080" // default for Docker
	}

	// Provider configuration
	aiProvider := os.Getenv("AI_PROVIDER")
	if aiProvider == "" {
		aiProvider = "local" // default to local Ollama
	}

	// OpenAI configuration
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiModel == "" {
		openaiModel = "gpt-4o-mini" // default model
	}
	openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseURL == "" {
		openaiBaseURL = "https://api.openai.com/v1"
	}

	// Groq configuration
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	groqModel := os.Getenv("GROQ_MODEL")
	if groqModel == "" {
		groqModel = "llama3-8b-8192" // default model
	}
	groqBaseURL := os.Getenv("GROQ_BASE_URL")
	if groqBaseURL == "" {
		groqBaseURL = "https://api.groq.com/openai/v1"
	}

	// Gemini configuration
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash" // default model
	}
	geminiBaseURL := os.Getenv("GEMINI_BASE_URL")
	if geminiBaseURL == "" {
		geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai" // default Gemini API URL
	}

	// Claude configuration
	claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
	claudeModel := os.Getenv("CLAUDE_MODEL")
	if claudeModel == "" {
		claudeModel = "claude-3-5-haiku-latest" // default model
	}
	claudeBaseURL := os.Getenv("CLAUDE_BASE_URL")
	if claudeBaseURL == "" {
		claudeBaseURL = "https://api.anthropic.com/v1" // default Claude API URL
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080" // default listen address (container healthcheck expects 8080)
	}

	return &Config{
		LogLevel:              parseLogLevel(os.Getenv("LOG_LEVEL")),
		ListenAddr:            listenAddr,
		PhoneNumber:           os.Getenv("SIGNAL_PHONE_NUMBER"),
		SignalURL:             signalURL,
		DatabasePath:          databasePath,
		LocalModel:            localModel,
		OllamaKeepAlive:       ollamaKeepAlive,
		OllamaHost:            ollamaHost,
		SummarizationInterval: summarizationInterval,

		AIProvider: aiProvider,

		OpenAIAPIKey:  openaiAPIKey,
		OpenAIModel:   openaiModel,
		OpenAIBaseURL: openaiBaseURL,

		GroqAPIKey:  groqAPIKey,
		GroqModel:   groqModel,
		GroqBaseURL: groqBaseURL,

		GeminiAPIKey:  geminiAPIKey,
		GeminiModel:   geminiModel,
		GeminiBaseURL: geminiBaseURL,

		ClaudeAPIKey:  claudeAPIKey,
		ClaudeModel:   claudeModel,
		ClaudeBaseURL: claudeBaseURL,
	}
}

func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
