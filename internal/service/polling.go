package service

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
	"uptime/internal/database"

	"github.com/robfig/cron/v3"
)

type cronEntry struct {
	id       cron.EntryID
	schedule string
}

// PollingService is the background worker that runs the health checks.
//
// It schedules one cron job per target using robfig/cron (in 6-field, seconds
// enabled format) and keeps the scheduler in sync with the database. A periodic
// reconciliation (resyncInterval, default 30s) picks up new, deleted, or
// rescheduled targets without a restart: each registered job's cron.EntryID and
// schedule are tracked in the entries map, so a changed schedule is removed and
// re-added rather than updated in place (robfig/cron has no edit method).
//
// Lifecycle:
//   - Start(ctx) performs an initial sync, starts the scheduler, then loops on a
//     ticker calling sync() every resyncInterval, stopping cleanly on ctx cancel.
//   - sync() reconciles the database with the scheduler in three passes:
//     remove entries for deleted targets, reschedule entries whose schedule
//     changed, and register new targets.
//   - addTarget(t) registers a single cron job whose closure pings that target.
//   - pingTarget(t) performs the actual check: it issues an HTTP GET with a 60s
//     timeout, measures latency, and records a Check row in the database (down
//     with error message on failure, otherwise with status code and response
//     time; isUp = statusCode < 500).
//
// Concurrency: the entries map is only mutated from the Start goroutine, and
// cron.Add/Remove are goroutine-safe, so no locking is required. pingTarget runs
// on the scheduler's goroutines and only touches the repository and HTTP client.
type PollingService struct {
	repo           *database.TargetRepository
	client         *http.Client
	cron           *cron.Cron
	entries        map[int]*cronEntry
	resyncInterval time.Duration
}

func NewPollingService(repo *database.TargetRepository) *PollingService {
	return &PollingService{
		repo: repo,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		cron:           cron.New(cron.WithSeconds()),
		entries:        make(map[int]*cronEntry),
		resyncInterval: 30 * time.Second,
	}
}

func (s *PollingService) Start(ctx context.Context) {
	if err := s.sync(); err != nil {
		log.Printf("[Polling] Error resyncing targets: %v", err)
		return
	}

	s.cron.Start()

	ticker := time.NewTicker(s.resyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.cron.Stop()
			return
		case <-ticker.C:
			if err := s.sync(); err != nil {
				log.Printf("[Polling] Error resyncing targets: %v", err)
			}
		}
	}
}

func (s *PollingService) addTarget(t database.Target) error {
	id, err := s.cron.AddFunc(t.Schedule, func() {
		s.pingTarget(t)
	})
	if err != nil {
		return err
	}
	s.entries[t.ID] = &cronEntry{id: id, schedule: t.Schedule}
	return nil
}

func (s *PollingService) sync() error {
	targets, err := s.repo.GetTargets()
	if err != nil {
		return err
	}

	valid := make(map[int]database.Target, len(targets))
	for _, t := range targets {
		valid[t.ID] = t
	}

	for id, entry := range s.entries {
		t, ok := valid[id]
		if !ok {
			s.cron.Remove(entry.id)
			delete(s.entries, id)
			log.Printf("[Polling] Removed cron for target %d", id)
			continue
		}
		if entry.schedule != t.Schedule {
			s.cron.Remove(entry.id)
			delete(s.entries, id)
			if err := s.addTarget(t); err != nil {
				log.Printf("[Polling] Error re-adding cron for target %d (schedule %s): %v", t.ID, t.Schedule, err)
			} else {
				log.Printf("[Polling] Rescheduled target %d: %q -> %q", t.ID, entry.schedule, t.Schedule)
			}
		}
	}

	for _, t := range targets {
		if _, ok := s.entries[t.ID]; ok {
			continue
		}
		if err := s.addTarget(t); err != nil {
			log.Printf("[Polling] Error adding cron for target %d (schedule %s): %v", t.ID, t.Schedule, err)
		} else {
			log.Printf("[Polling] Registered cron for target %d (schedule %s)", t.ID, t.Schedule)
		}
	}

	return nil
}

func (s *PollingService) pingTarget(t database.Target) {
	start := time.Now()
	resp, err := s.client.Get(t.URL)
	duration := time.Since(start).Milliseconds()
	durInt := int(duration)

	if err != nil {
		errMsg := err.Error()
		log.Printf("[Polling] Target %s (%s) - Error: %v (took %dms)", t.Name, t.URL, err, duration)
		_ = s.repo.CreateCheck(&database.Check{
			TargetID:       t.ID,
			ResponseTimeMS: &durInt,
			IsUp:           false,
			ErrorMessage:   &errMsg,
		})
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("[Polling] Error closing response body for target %s (%s): %v", t.Name, t.URL, err)
		}
	}(resp.Body)

	statusCode := resp.StatusCode
	isUp := statusCode < 500

	if isUp {
		log.Printf("[Polling] Target %s (%s) - Reachable: status %d, took %dms", t.Name, t.URL, statusCode, duration)
	} else {
		log.Printf("[Polling] Target %s (%s) - Unreachable (Server Error): status %d, took %dms", t.Name, t.URL, statusCode, duration)
	}

	_ = s.repo.CreateCheck(&database.Check{
		TargetID:       t.ID,
		StatusCode:     &statusCode,
		ResponseTimeMS: &durInt,
		IsUp:           isUp,
	})
}
