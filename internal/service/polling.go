package service

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"uptime/internal/database"
)

type PollingService struct {
	repo   *database.TargetRepository
	client *http.Client
}

func NewPollingService(repo *database.TargetRepository) *PollingService {
	return &PollingService{
		repo: repo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *PollingService) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	go s.PollTargets()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go s.PollTargets()
		}
	}
}

func (s *PollingService) PollTargets() {
	targets, err := s.repo.GetTargets()
	if err != nil {
		log.Printf("[Polling] Error fetching targets: %v", err)
		return
	}

	if len(targets) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t database.Target) {
			defer wg.Done()
			s.pingTarget(t)
		}(target)
	}
	wg.Wait()
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
