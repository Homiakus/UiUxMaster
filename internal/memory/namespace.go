package memory

import (
	"errors"
	"fmt"
	"strings"
)

// Standard namespace prefixes
const (
	PrefixKnowledgeGlobal  = "knowledge/global-design"
	PrefixKnowledgeProject = "knowledge/project/"
	PrefixEvidenceProject  = "evidence/project/"
	PrefixResearchGlobal   = "research/global"
	PrefixSkillMeta        = "skillmeta/"
)

var (
	ErrInvalidNamespace   = errors.New("invalid memory namespace")
	ErrUnauthorizedAccess = errors.New("unauthorized cross-namespace access")
)

// Namespace represents a validated, structured memory partition.
type Namespace struct {
	raw       string
	category  string
	projectID string
	skillID   string
}

// NewGlobalDesignNamespace returns the global design knowledge namespace.
func NewGlobalDesignNamespace() Namespace {
	return Namespace{
		raw:      PrefixKnowledgeGlobal,
		category: "knowledge/global",
	}
}

// NewProjectKnowledgeNamespace returns a project-scoped knowledge namespace.
func NewProjectKnowledgeNamespace(projectID string) (Namespace, error) {
	if strings.TrimSpace(projectID) == "" {
		return Namespace{}, fmt.Errorf("%w: projectID cannot be empty", ErrInvalidNamespace)
	}
	cleanID := strings.TrimSpace(projectID)
	return Namespace{
		raw:       PrefixKnowledgeProject + cleanID,
		category:  "knowledge/project",
		projectID: cleanID,
	}, nil
}

// NewProjectEvidenceNamespace returns a project-scoped evidence namespace.
func NewProjectEvidenceNamespace(projectID string) (Namespace, error) {
	if strings.TrimSpace(projectID) == "" {
		return Namespace{}, fmt.Errorf("%w: projectID cannot be empty", ErrInvalidNamespace)
	}
	cleanID := strings.TrimSpace(projectID)
	return Namespace{
		raw:       PrefixEvidenceProject + cleanID,
		category:  "evidence/project",
		projectID: cleanID,
	}, nil
}

// NewResearchGlobalNamespace returns the global research namespace.
func NewResearchGlobalNamespace() Namespace {
	return Namespace{
		raw:      PrefixResearchGlobal,
		category: "research/global",
	}
}

// NewSkillMetaNamespace returns a skill-scoped metadata namespace.
func NewSkillMetaNamespace(skillID string) (Namespace, error) {
	if strings.TrimSpace(skillID) == "" {
		return Namespace{}, fmt.Errorf("%w: skillID cannot be empty", ErrInvalidNamespace)
	}
	cleanID := strings.TrimSpace(skillID)
	return Namespace{
		raw:      PrefixSkillMeta + cleanID,
		category: "skillmeta",
		skillID:  cleanID,
	}, nil
}

// ParseNamespace validates and parses a raw namespace string.
func ParseNamespace(raw string) (Namespace, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Namespace{}, fmt.Errorf("%w: empty namespace", ErrInvalidNamespace)
	}

	if trimmed == PrefixKnowledgeGlobal {
		return NewGlobalDesignNamespace(), nil
	}
	if trimmed == PrefixResearchGlobal {
		return NewResearchGlobalNamespace(), nil
	}
	if strings.HasPrefix(trimmed, PrefixKnowledgeProject) {
		projectID := strings.TrimPrefix(trimmed, PrefixKnowledgeProject)
		return NewProjectKnowledgeNamespace(projectID)
	}
	if strings.HasPrefix(trimmed, PrefixEvidenceProject) {
		projectID := strings.TrimPrefix(trimmed, PrefixEvidenceProject)
		return NewProjectEvidenceNamespace(projectID)
	}
	if strings.HasPrefix(trimmed, PrefixSkillMeta) {
		skillID := strings.TrimPrefix(trimmed, PrefixSkillMeta)
		return NewSkillMetaNamespace(skillID)
	}

	return Namespace{}, fmt.Errorf("%w: unknown prefix %q", ErrInvalidNamespace, trimmed)
}

// String returns the canonical raw namespace identifier.
func (n Namespace) String() string {
	return n.raw
}

// Category returns the broad namespace category.
func (n Namespace) Category() string {
	return n.category
}

// ProjectID returns the project ID if project-scoped.
func (n Namespace) ProjectID() string {
	return n.projectID
}

// SkillID returns the skill ID if skillmeta-scoped.
func (n Namespace) SkillID() string {
	return n.skillID
}

// IsProjectPrivate returns true if this namespace contains project-private data.
func (n Namespace) IsProjectPrivate() bool {
	return n.projectID != ""
}

// IsGlobal returns true if this namespace is globally shared.
func (n Namespace) IsGlobal() bool {
	return n.raw == PrefixKnowledgeGlobal || n.raw == PrefixResearchGlobal
}

// CanAccess checks if a request with a given scope is permitted to access targetNamespace under the firewall.
func CanAccess(requestScope Namespace, targetNamespace Namespace) bool {
	// Global namespaces are accessible by any request.
	if targetNamespace.IsGlobal() {
		return true
	}

	// Project-private namespaces can only be accessed by requests for the exact same project.
	if targetNamespace.IsProjectPrivate() {
		return requestScope.projectID != "" && requestScope.projectID == targetNamespace.projectID
	}

	// Skillmeta can only be accessed by matching skill scope or global.
	if targetNamespace.skillID != "" {
		if requestScope.IsGlobal() {
			return true
		}
		return requestScope.skillID == targetNamespace.skillID
	}

	return false
}
