package ai

import (
	"context"
	"log/slog"
	"summarizarr/internal/database"
	"time"
)

// Scheduler is a scheduler for the AI summarization service.
type Scheduler struct {
	db       DB
	aiClient *Client
	interval time.Duration
}

// DB is the interface for the database.
type DB interface {
	GetMessagesForSummarization(groupID int64, start, end int64) ([]database.MessageForSummary, error)
	GetGroups() ([]int64, error)
	SaveSummary(groupID int64, summaryText string, start, end int64) error
	GetUserNameByID(userID int64) (string, error)
	GetGroupNameByID(groupID int64) (string, error)
}

// NewScheduler creates a new scheduler.
func NewScheduler(db DB, aiClient *Client, interval time.Duration) *Scheduler {
	return &Scheduler{
		db:       db,
		aiClient: aiClient,
		interval: interval,
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runSummarization(ctx)
		}
	}
}

func (s *Scheduler) runSummarization(ctx context.Context) {
	slog.Info("Running summarization...")

	groups, err := s.db.GetGroups()
	if err != nil {
		slog.Error("Error getting groups for summarization", "error", err)
		return
	}

	for _, groupID := range groups {
		go s.summarizeGroup(ctx, groupID)
	}
}

func (s *Scheduler) summarizeGroup(ctx context.Context, groupID int64) {
	// Use millisecond timestamps to match message storage format
	endMs := time.Now().UnixMilli()
	startMs := time.Now().Add(-s.interval).UnixMilli()

	slog.Debug("Summarizing group", "group_id", groupID, "start_ms", startMs, "end_ms", endMs)

	messages, err := s.db.GetMessagesForSummarization(groupID, startMs, endMs)
	if err != nil {
		slog.Error("Error getting messages for summarization", "group_id", groupID, "error", err)
		return
	}

	if len(messages) == 0 {
		slog.Debug("No messages found for summarization", "group_id", groupID)
		return
	}

	slog.Info("Generating summary", "group_id", groupID, "message_count", len(messages))

	summary, err := s.aiClient.Summarize(ctx, messages)
	if err != nil {
		slog.Error("Error summarizing messages", "group_id", groupID, "error", err)
		return
	}

	if err := s.db.SaveSummary(groupID, summary, startMs, endMs); err != nil {
		slog.Error("Error saving summary", "group_id", groupID, "error", err)
		return
	}

	slog.Info("Saved summary", "group_id", groupID, "summary_length", len(summary))
}
