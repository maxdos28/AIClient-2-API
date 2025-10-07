package models

// Provider weight for load balancing
type ProviderWeight struct {
	Provider Provider
	Weight   int
}