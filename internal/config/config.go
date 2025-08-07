package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	LogLevel              slog.Level
	PhoneNumber           string
	DatabasePath          string
	AIBackend             string
	LocalModel            string
	OllamaAutoDownload    bool
	OllamaKeepAlive       string
	OllamaHost            string
	ModelsPath            string
	SummarizationInterval string
	OpenAIAPIKey          string
	OpenAIModel           string
}

// New creates a new Config from environment variables.
func New() *Config {
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "summarizarr.db" // default path
	}

	modelsPath := os.Getenv("MODELS_PATH")
	if modelsPath == "" {
		modelsPath = "./models" // default path
	}

	aiBackend := os.Getenv("AI_BACKEND")
	if aiBackend == "" {
		aiBackend = "local" // always local for now
	}

	localModel := os.Getenv("LOCAL_MODEL")
	if localModel == "" {
		localModel = "llama3.2:1b" // default model - smaller memory footprint
	}

	ollamaAutoDownload := true
	if val := os.Getenv("OLLAMA_AUTO_DOWNLOAD"); val == "false" {
		ollamaAutoDownload = false
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

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiModel == "" {
		openaiModel = "gpt-4o" // default model
	}

	return &Config{
		LogLevel:              parseLogLevel(os.Getenv("LOG_LEVEL")),
		PhoneNumber:           os.Getenv("SIGNAL_PHONE_NUMBER"),
		DatabasePath:          databasePath,
		AIBackend:             aiBackend,
		LocalModel:            localModel,
		OllamaAutoDownload:    ollamaAutoDownload,
		OllamaKeepAlive:       ollamaKeepAlive,
		OllamaHost:            ollamaHost,
		ModelsPath:            modelsPath,
		SummarizationInterval: summarizationInterval,
		OpenAIAPIKey:          openaiAPIKey,
		OpenAIModel:           openaiModel,
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
