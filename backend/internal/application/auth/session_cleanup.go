package auth

import (
	"log"
	"time"
)

// SessionCleanupJob periodically cleans up expired sessions
type SessionCleanupJob struct {
	sessionRepo *SessionRepository
	interval    time.Duration
	stopCh      chan struct{}
}

// NewSessionCleanupJob creates a new session cleanup job
func NewSessionCleanupJob(sessionRepo *SessionRepository, interval time.Duration) *SessionCleanupJob {
	if interval == 0 {
		interval = 1 * time.Hour // Default: run every hour
	}
	return &SessionCleanupJob{
		sessionRepo: sessionRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the periodic cleanup job
func (j *SessionCleanupJob) Start() {
	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		log.Printf("[SessionCleanup] Started, running every %v", j.interval)

		for {
			select {
			case <-ticker.C:
				j.runCleanup()
			case <-j.stopCh:
				log.Println("[SessionCleanup] Stopped")
				return
			}
		}
	}()
}

// Stop stops the cleanup job
func (j *SessionCleanupJob) Stop() {
	close(j.stopCh)
}

// runCleanup performs a single cleanup operation
func (j *SessionCleanupJob) runCleanup() {
	if err := j.sessionRepo.CleanupExpiredSessions(); err != nil {
		log.Printf("[SessionCleanup] Error cleaning up sessions: %v", err)
	} else {
		log.Println("[SessionCleanup] Cleanup completed successfully")
	}
}
