package database

type TargetRepository struct{}

func NewTargetRepository() *TargetRepository {
	return &TargetRepository{}
}

func (r *TargetRepository) GetTargets() string {
	return "Hello world"
}
