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

type PollingService struct {
	repo   *database.TargetRepository
	client *http.Client
	cron   *cron.Cron
}

func NewPollingService(repo *database.TargetRepository) *PollingService {
	return &PollingService{
		repo: repo,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		cron: cron.New(cron.WithSeconds()),
	}
}

func (s *PollingService) Start(ctx context.Context) {
	targets, err := s.repo.GetTargets()
	if err != nil {
		log.Printf("[Polling] Error fetching targets: %v", err)
		return
	}

	for _, target := range targets {
		t := target
		_, err := s.cron.AddFunc(t.Schedule, func() {
			s.pingTarget(t)
		})
		if err != nil {
			log.Printf("[Polling] Error adding cron for target %s (schedule %s): %v", t.Name, t.Schedule, err)
		}
	}

	s.cron.Start()

	<-ctx.Done()
	s.cron.Stop()
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
			log.Fatal(err)
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
