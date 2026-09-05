package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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
	ErrScopeRequired      = errors.New("memory scope is required")
	ErrAdmissionRoute     = errors.New("memory admission route is not authorized")
)

// Namespace is a validated structured partition. Its canonical JSON form is the
// raw namespace string; keeping identity serializable is mandatory for durable
// provenance and future stores.
type Namespace struct {
	raw       string
	category  string
	projectID string
	skillID   string
}

func NewGlobalDesignNamespace() Namespace {
	return Namespace{raw: PrefixKnowledgeGlobal, category: "knowledge/global"}
}

func NewProjectKnowledgeNamespace(projectID string) (Namespace, error) {
	cleanID := strings.TrimSpace(projectID)
	if cleanID == "" || strings.ContainsAny(cleanID, "\r\n\x00") {
		return Namespace{}, fmt.Errorf("%w: invalid projectID", ErrInvalidNamespace)
	}
	return Namespace{raw: PrefixKnowledgeProject + cleanID, category: "knowledge/project", projectID: cleanID}, nil
}

func NewProjectEvidenceNamespace(projectID string) (Namespace, error) {
	cleanID := strings.TrimSpace(projectID)
	if cleanID == "" || strings.ContainsAny(cleanID, "\r\n\x00") {
		return Namespace{}, fmt.Errorf("%w: invalid projectID", ErrInvalidNamespace)
	}
	return Namespace{raw: PrefixEvidenceProject + cleanID, category: "evidence/project", projectID: cleanID}, nil
}

func NewResearchGlobalNamespace() Namespace {
	return Namespace{raw: PrefixResearchGlobal, category: "research/global"}
}

func NewSkillMetaNamespace(skillID string) (Namespace, error) {
	cleanID := strings.TrimSpace(skillID)
	if cleanID == "" || strings.ContainsAny(cleanID, "\r\n\x00") {
		return Namespace{}, fmt.Errorf("%w: invalid skillID", ErrInvalidNamespace)
	}
	return Namespace{raw: PrefixSkillMeta + cleanID, category: "skillmeta", skillID: cleanID}, nil
}

func ParseNamespace(raw string) (Namespace, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Namespace{}, fmt.Errorf("%w: empty namespace", ErrInvalidNamespace)
	}
	if trimmed == PrefixKnowledgeGlobal { return NewGlobalDesignNamespace(), nil }
	if trimmed == PrefixResearchGlobal { return NewResearchGlobalNamespace(), nil }
	if strings.HasPrefix(trimmed, PrefixKnowledgeProject) {
		return NewProjectKnowledgeNamespace(strings.TrimPrefix(trimmed, PrefixKnowledgeProject))
	}
	if strings.HasPrefix(trimmed, PrefixEvidenceProject) {
		return NewProjectEvidenceNamespace(strings.TrimPrefix(trimmed, PrefixEvidenceProject))
	}
	if strings.HasPrefix(trimmed, PrefixSkillMeta) {
		return NewSkillMetaNamespace(strings.TrimPrefix(trimmed, PrefixSkillMeta))
	}
	return Namespace{}, fmt.Errorf("%w: unknown prefix %q", ErrInvalidNamespace, trimmed)
}

func (n Namespace) String() string { return n.raw }
func (n Namespace) Category() string { return n.category }
func (n Namespace) ProjectID() string { return n.projectID }
func (n Namespace) SkillID() string { return n.skillID }
func (n Namespace) IsProjectPrivate() bool { return n.projectID != "" }
func (n Namespace) IsGlobal() bool { return n.raw == PrefixKnowledgeGlobal || n.raw == PrefixResearchGlobal }
func (n Namespace) IsValid() bool { _, err := ParseNamespace(n.raw); return err == nil }
func (n Namespace) Equal(other Namespace) bool { return n.raw != "" && n.raw == other.raw }

func (n Namespace) MarshalJSON() ([]byte, error) {
	if n.raw == "" { return []byte("null"), nil }
	if !n.IsValid() { return nil, ErrInvalidNamespace }
	return json.Marshal(n.raw)
}

func (n *Namespace) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*n = Namespace{}
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil { return err }
	parsed, err := ParseNamespace(raw)
	if err != nil { return err }
	*n = parsed
	return nil
}

// CanAccess is retrieval authorization. A project may read its own private
// partitions plus global knowledge. A global request never gains project-private
// visibility. Missing request scope is denied by callers before this function.
func CanAccess(requestScope Namespace, targetNamespace Namespace) bool {
	if !requestScope.IsValid() || !targetNamespace.IsValid() { return false }
	if targetNamespace.IsGlobal() { return true }
	if targetNamespace.IsProjectPrivate() {
		return requestScope.projectID != "" && requestScope.projectID == targetNamespace.projectID
	}
	if targetNamespace.skillID != "" {
		return requestScope.skillID != "" && requestScope.skillID == targetNamespace.skillID
	}
	return false
}

// CanAdmitOrdinary is the write firewall for non-promotion admission. Ordinary
// admission may move evidence->knowledge within the same project, but it can
// never broaden visibility. Global data stays in the exact same global
// namespace; skill metadata stays with the same skill.
func CanAdmitOrdinary(source, target Namespace) bool {
	if !source.IsValid() || !target.IsValid() { return false }
	if source.IsProjectPrivate() {
		return target.IsProjectPrivate() && source.ProjectID() == target.ProjectID()
	}
	if source.IsGlobal() {
		return source.Equal(target)
	}
	if source.SkillID() != "" {
		return target.SkillID() == source.SkillID() && source.Equal(target)
	}
	return false
}
