package service

import "uptime/internal/database"

type IncidentService struct {
	repo *database.TargetRepository
}

func NewIncidentService(repo *database.TargetRepository) *IncidentService {
	return &IncidentService{repo: repo}
}

func (s *IncidentService) GetIncidents() ([]database.Incident, error) {
	return s.repo.GetIncidents()
}

func (s *IncidentService) GetLatestIncident(targetID int, incidentType string) (*database.Incident, error) {
	return s.repo.GetLatestIncident(targetID, incidentType)
}

func (s *IncidentService) MarkRead(id int) error {
	return s.repo.MarkIncidentRead(id)
}

func (s *IncidentService) MarkAllRead() error {
	return s.repo.MarkAllIncidentsRead()
}
