package config

type Repository struct {
	Workflows map[string]*Workflow `yaml:"workflows" json:"workflows"`
}
