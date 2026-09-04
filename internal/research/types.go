package research

import (
	"errors"
	"time"
)

var (
	ErrResearchUnavailable = errors.New("research: sidecar provider unavailable")
	ErrInvalidResearchReq  = errors.New("research: invalid query or parameters")
	ErrInvalidBundle       = errors.New("research: malformed research bundle")
)

// ResearchRequest parameters for querying design standards, tokens, or best practices.
type ResearchRequest struct {
	Query      string        `json:"query"`
	Domain     string        `json:"domain,omitempty"` // e.g. "wcag22", "editorial", "typography"
	MaxSources int           `json:"max_sources,omitempty"`
	MaxAge     time.Duration `json:"max_age,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
}

// ResearchSourceRef identifies an authoritative source citation.
type ResearchSourceRef struct {
	URI             string   `json:"uri"`
	Title           string   `json:"title"`
	Authors         []string `json:"authors,omitempty"`
	PublicationDate string   `json:"publication_date,omitempty"`
	Summary         string   `json:"summary"`
	Digest          string   `json:"digest"`
	Reliability     float64  `json:"reliability"` // 0.0 to 1.0
}

// Claim represents a structured assertion extracted from research sources.
type Claim struct {
	ID         string  `json:"id"`
	Subject    string  `json:"subject"`
	Predicate  string  `json:"predicate"`
	Object     string  `json:"object"`
	SourceURI  string  `json:"source_uri"`
	Confidence float64 `json:"confidence"`
	Context    string  `json:"context,omitempty"`
}

// ResearchBundle represents the canonical output package from a research operation.
type ResearchBundle struct {
	BundleID  string              `json:"bundle_id"`
	Query     string              `json:"query"`
	Domain    string              `json:"domain,omitempty"`
	FetchedAt time.Time           `json:"fetched_at"`
	Sources   []ResearchSourceRef `json:"sources"`
	Claims    []Claim             `json:"claims"`
	Digest    string              `json:"digest"`
}
