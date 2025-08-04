package ai

import (
	"context"
	"log"
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
	log.Println("Running summarization...")

	groups, err := s.db.GetGroups()
	if err != nil {
		log.Printf("Error getting groups for summarization: %v", err)
		return
	}

	for _, groupID := range groups {
		go s.summarizeGroup(ctx, groupID)
	}
}

func (s *Scheduler) summarizeGroup(ctx context.Context, groupID int64) {
	end := time.Now().Unix()
	start := time.Now().Add(-s.interval).Unix()

	messages, err := s.db.GetMessagesForSummarization(groupID, start, end)
	if err != nil {
		log.Printf("Error getting messages for summarization for group %d: %v", groupID, err)
		return
	}

	if len(messages) == 0 {
		return
	}

	summary, err := s.aiClient.Summarize(ctx, messages)
	if err != nil {
		log.Printf("Error summarizing messages for group %d: %v", groupID, err)
		return
	}

	if err := s.db.SaveSummary(groupID, summary, start, end); err != nil {
		log.Printf("Error saving summary for group %d: %v", groupID, err)
		return
	}

	log.Printf("Saved summary for group %d", groupID)
}