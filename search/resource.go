package search

// Resource is the resource object returned from elasticsearch
type Resource struct {
	Account    string
	CreatedBy  string `json:"yale:created_by"`
	ID         string
	Name       string `json:"omitempty"`
	Provider   string
	Status     string
	RenewedAt  string `json:"yale:renewed_at, omitempty"`
	NotifiedAt string `josn:"yale:notified_at, omitempty"`
}
