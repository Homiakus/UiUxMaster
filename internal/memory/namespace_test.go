package memory

import (
	"testing"
)

func TestNamespaceValidationAndFirewall(t *testing.T) {
	global := NewGlobalDesignNamespace()
	if !global.IsGlobal() {
		t.Fatalf("expected global namespace")
	}
	if global.String() != PrefixKnowledgeGlobal {
		t.Fatalf("expected raw prefix %s, got %s", PrefixKnowledgeGlobal, global.String())
	}

	projKnow, err := NewProjectKnowledgeNamespace("proj_alpha")
	if err != nil {
		t.Fatalf("unexpected error creating project knowledge: %v", err)
	}
	if !projKnow.IsProjectPrivate() {
		t.Fatalf("expected project private namespace")
	}
	if projKnow.ProjectID() != "proj_alpha" {
		t.Fatalf("expected projectID proj_alpha, got %s", projKnow.ProjectID())
	}

	projEv, err := NewProjectEvidenceNamespace("proj_alpha")
	if err != nil {
		t.Fatalf("unexpected error creating project evidence: %v", err)
	}
	if projEv.ProjectID() != "proj_alpha" {
		t.Fatalf("expected proj_alpha, got %s", projEv.ProjectID())
	}

	skill, err := NewSkillMetaNamespace("motion-ux")
	if err != nil {
		t.Fatalf("unexpected error creating skillmeta: %v", err)
	}
	if skill.SkillID() != "motion-ux" {
		t.Fatalf("expected motion-ux, got %s", skill.SkillID())
	}

	// Parsing tests
	parsedGlobal, err := ParseNamespace("knowledge/global-design")
	if err != nil || !parsedGlobal.IsGlobal() {
		t.Fatalf("failed parsing global namespace: %v", err)
	}

	parsedProj, err := ParseNamespace("knowledge/project/client_x")
	if err != nil || parsedProj.ProjectID() != "client_x" {
		t.Fatalf("failed parsing project namespace: %v", err)
	}

	parsedEv, err := ParseNamespace("evidence/project/client_x")
	if err != nil || parsedEv.ProjectID() != "client_x" {
		t.Fatalf("failed parsing evidence namespace: %v", err)
	}

	parsedSkill, err := ParseNamespace("skillmeta/vlm-critic")
	if err != nil || parsedSkill.SkillID() != "vlm-critic" {
		t.Fatalf("failed parsing skillmeta namespace: %v", err)
	}

	_, err = ParseNamespace("invalid/namespace/path")
	if err == nil {
		t.Fatalf("expected error for invalid namespace")
	}

	// Firewall access checks
	projBeta, _ := NewProjectKnowledgeNamespace("proj_beta")

	// Global can be accessed by anyone
	if !CanAccess(projKnow, global) {
		t.Fatalf("expected access to global from project scope")
	}

	// Same project can access own knowledge
	if !CanAccess(projKnow, projEv) {
		t.Fatalf("expected access to same project evidence")
	}

	// Cross-project leakage MUST be blocked
	if CanAccess(projBeta, projKnow) {
		t.Fatalf("firewall violation: proj_beta accessed proj_alpha knowledge")
	}
	if CanAccess(projBeta, projEv) {
		t.Fatalf("firewall violation: proj_beta accessed proj_alpha evidence")
	}
}
