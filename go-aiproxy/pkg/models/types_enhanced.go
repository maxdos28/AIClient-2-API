package models

// Additional provider constants
const (
	ProviderKiro Provider = "kiro"
	ProviderQwen Provider = "qwen"
)

// Provider weight for load balancing
type ProviderWeight struct {
	Provider Provider
	Weight   int
}