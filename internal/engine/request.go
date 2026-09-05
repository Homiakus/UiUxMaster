package engine

import (
	"context"
	"fmt"
	"image"
	"sort"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
)

// ValidationRequest is the protocol-independent input boundary for UI/UX validation.
// Callers (MCP, Axiom, CLI, tests) supply changes, intent, and optional overrides,
// and the engine orchestrates scope resolution and evidence planning.
type ValidationRequest struct {
	RunID             string                       `json:"run_id"`
	ProjectID         string                       `json:"project_id,omitempty"`
	SourceDigest      string                       `json:"source_digest,omitempty"`
	ChangedFiles      []string                     `json:"changed_files,omitempty"`
	ChangedTokens     []string                     `json:"changed_tokens,omitempty"`
	ChangedNodes      []string                     `json:"changed_nodes,omitempty"`
	Intent            evidenceplan.Intent          `json:"intent,omitempty"`
	FinalGate         bool                         `json:"final_gate,omitempty"`
	RequireLegalPass  bool                         `json:"require_legal_pass,omitempty"`
	Scope             invalidation.ValidationScope `json:"scope,omitempty"`
	ForceWholeSite    bool                         `json:"force_whole_site,omitempty"`
	TargetRoutes      []string                     `json:"target_routes,omitempty"`
	Viewports         []string                     `json:"viewports,omitempty"`
	Themes            []string                     `json:"themes,omitempty"`
	Need              EvidenceNeed                 `json:"need,omitempty"`
	Region            *evidenceplan.Region         `json:"region,omitempty"`
	HTML              []byte                       `json:"html,omitempty"`
	CSS               []byte                       `json:"css,omitempty"`
	BaseURL           string                       `json:"base_url,omitempty"`
	BaselineRGBA      *image.RGBA                  `json:"-"`
	Tolerance         uint8                        `json:"tolerance,omitempty"`
}

// Normalize ensures deterministic slices and sensible default values.
func (r *ValidationRequest) Normalize() {
	if r.RunID == "" {
		r.RunID = "run:default"
	}
	if r.Intent == "" && r.Need == (EvidenceNeed{}) {
		r.Intent = evidenceplan.IntentQuickStructural
	}
	if r.FinalGate {
		r.RequireLegalPass = true
	}
	r.ChangedFiles = uniqueSorted(r.ChangedFiles)
	r.ChangedTokens = uniqueSorted(r.ChangedTokens)
	r.ChangedNodes = uniqueSorted(r.ChangedNodes)
	r.TargetRoutes = uniqueSorted(r.TargetRoutes)
	r.Viewports = uniqueSorted(r.Viewports)
	r.Themes = uniqueSorted(r.Themes)
}

// DeriveNeed computes the EvidenceNeed based on requested intent, final gate, and explicit needs.
func (r *ValidationRequest) DeriveNeed() EvidenceNeed {
	need := r.Need

	if r.FinalGate {
		need.Geometry = true
		need.Styles = true
		need.CleanState = true
	}

	if r.Region != nil {
		need.Pixels = true
	}

	switch r.Intent {
	case evidenceplan.IntentQuickStructural:
		need.Geometry = true
	case evidenceplan.IntentInteraction:
		need.Geometry = true
		need.Styles = true
		need.Scenario = true
	case evidenceplan.IntentTypography:
		need.Geometry = true
		need.Styles = true
	case evidenceplan.IntentFullDeterministic:
		need.Geometry = true
		need.Styles = true
	case evidenceplan.IntentVisualRegion:
		need.Geometry = true
		need.Styles = true
		need.Pixels = true
	}

	return need
}

func isFontToken(token string) bool {
	lower := strings.ToLower(token)
	return strings.Contains(lower, "font") || strings.Contains(lower, "family") || strings.Contains(lower, "type") || strings.Contains(lower, "typo")
}

func hasFontToken(tokens []string) bool {
	for _, t := range tokens {
		if isFontToken(t) {
			return true
		}
	}
	return false
}

// Signals converts the request into evidenceplan.Signals for evidence shape synthesis.
func (r *ValidationRequest) Signals(risk fidelity.RiskLevel) evidenceplan.Signals {
	return evidenceplan.Signals{
		Intent:             r.Intent,
		Risk:               risk,
		CustomFontsChanged: hasFontToken(r.ChangedTokens),
		SemanticsChanged:   len(r.ChangedNodes) > 0,
		InteractionChanged: r.Need.Scenario,
		RuntimeChanged:     r.FinalGate,
		FinalGate:          r.FinalGate,
		Region:             r.Region,
	}
}

// PlanScope coordinates impact analysis and invalidation policy to compute the authoritative
// ValidationScope for the request without requiring any protocol-specific types.
func PlanScope(ctx context.Context, req ValidationRequest, resolver *impact.Resolver, policy *invalidation.Policy) (ValidationRequest, invalidation.ValidationScope, error) {
	if err := ctx.Err(); err != nil {
		return ValidationRequest{}, invalidation.ValidationScope{}, err
	}

	req.Normalize()
	req.Need = req.DeriveNeed()

	if policy == nil {
		policy = invalidation.DefaultPolicy()
	}

	// Prepare change set from changed files, tokens, and nodes
	nodeIDs := make([]string, 0, len(req.ChangedFiles)+len(req.ChangedTokens)+len(req.ChangedNodes))
	for _, f := range req.ChangedFiles {
		nodeIDs = append(nodeIDs, "file:"+f)
	}
	for _, t := range req.ChangedTokens {
		nodeIDs = append(nodeIDs, "token:"+t)
	}
	nodeIDs = append(nodeIDs, req.ChangedNodes...)
	nodeIDs = uniqueSorted(nodeIDs)

	var impactSet impact.ImpactSet
	if resolver != nil && len(nodeIDs) > 0 {
		var err error
		impactSet, err = resolver.ApplyChanges(ctx, impact.ChangeSet{
			NodeIDs: nodeIDs,
		})
		if err != nil {
			return ValidationRequest{}, invalidation.ValidationScope{}, fmt.Errorf("engine: apply changes: %w", err)
		}
	} else {
		// Fallback minimal impact set if no graph resolver is provided
		impactSet = impact.ImpactSet{
			NodeIDs: nodeIDs,
			Broad:   req.ForceWholeSite,
		}
	}

	opts := invalidation.Options{
		ForceWholeSite: req.ForceWholeSite,
		ForceRoutes:    req.TargetRoutes,
		ForceViewports: req.Viewports,
		ForceThemes:    req.Themes,
	}

	scope := policy.Invalidate(impactSet, opts)
	req.Scope = scope
	return req, scope, nil
}

// PlanProjectScope connects a ProjectIndex directly to the ValidationRequest, resolving
// the authoritative ValidationScope from indexed repository files.
func PlanProjectScope(ctx context.Context, req ValidationRequest, index *impact.ProjectIndex, policy *invalidation.Policy) (ValidationRequest, invalidation.ValidationScope, error) {
	if err := ctx.Err(); err != nil {
		return ValidationRequest{}, invalidation.ValidationScope{}, err
	}
	if index == nil || index.Graph == nil {
		return ValidationRequest{}, invalidation.ValidationScope{}, fmt.Errorf("engine: project index or graph is nil")
	}

	req.Normalize()
	req.Need = req.DeriveNeed()

	if policy == nil {
		policy = invalidation.DefaultPolicy()
	}

	opts := invalidation.Options{
		ForceWholeSite: req.ForceWholeSite,
		ForceRoutes:    req.TargetRoutes,
		ForceViewports: req.Viewports,
		ForceThemes:    req.Themes,
	}

	scope, err := invalidation.ResolveProjectScope(ctx, index, req.ChangedFiles, policy, opts)
	if err != nil {
		return ValidationRequest{}, invalidation.ValidationScope{}, fmt.Errorf("engine: resolve project scope: %w", err)
	}

	req.Scope = scope
	return req, scope, nil
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v != "" {
			set[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
