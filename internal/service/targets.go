package service

import "uptime/internal/database"

type TargetService struct {
	repo *database.TargetRepository
}

func NewTargetService(repo *database.TargetRepository) *TargetService {
	return &TargetService{repo: repo}
}

func (s *TargetService) GetTargets() ([]database.Target, error) {
	return s.repo.GetTargets()
}

func (s *TargetService) GetTargetsWithRecentChecks(limit int) ([]database.TargetWithChecks, error) {
	return s.repo.GetTargetsWithRecentChecks(limit)
}

// UpdateTarget updates the name and schedule of an existing target and returns
// the updated target.
func (s *TargetService) UpdateTarget(id int, name string, schedule string) (*database.Target, error) {
	if err := s.repo.UpdateTarget(id, name, schedule); err != nil {
		return nil, err
	}
	return s.repo.GetTargetByID(id)
}
