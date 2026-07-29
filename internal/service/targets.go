package service

import "uptime/internal/database"

type TargetService struct {
	repo *database.TargetRepository
}

func NewTargetService(repo *database.TargetRepository) *TargetService {
	return &TargetService{repo: repo}
}

func (s *TargetService) GetTargets() string {
	return s.repo.GetTargets()
}
