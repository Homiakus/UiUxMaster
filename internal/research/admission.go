package research

import (
	"context"
	"fmt"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

// AdmissionAdapter converts ResearchBundles into SncSinCore memory admission candidates.
type AdmissionAdapter struct {
	store memory.Store
}

func NewAdmissionAdapter(store memory.Store) *AdmissionAdapter {
	return &AdmissionAdapter{store: store}
}

// MapBundleToAdmission converts research sources and claims into memory atoms and relationship edges.
// Research ingestion is ordinary admission into the same scope. Moving research
// claims from a project/global research partition into global design knowledge is
// a separate promotion operation owned by the memory promotion authority.
func (a *AdmissionAdapter) MapBundleToAdmission(bundle *ResearchBundle, targetScope memory.Namespace) (*memory.AdmissionBundle, error) {
	if bundle == nil {
		return nil, ErrInvalidBundle
	}

	ns := targetScope
	if !ns.IsValid() {
		ns = memory.NewResearchGlobalNamespace()
	}
	projectScope := "global"
	if ns.IsProjectPrivate() {
		projectScope = ns.ProjectID()
	}

	var atoms []memory.MemoryAtom
	var edges []memory.MemoryEdge
	now := time.Now()

	prov := memory.ProvenanceRecord{
		RunID:           fmt.Sprintf("research_%s", bundle.BundleID),
		EvidenceDigest:  bundle.Digest,
		Renderer:        "research_plane",
		Tier:            fidelity.TierL0,
		Environment:     "standards_catalog",
		RuleVersion:     "research.v1",
		CriticVersion:   "research_adapter",
		ProjectScope:    projectScope,
		SourceNamespace: ns.String(),
		Timestamp:       now,
		Outcome:         "ADMITTED",
	}

	sourceMap := make(map[string]string)
	for _, src := range bundle.Sources {
		sourceID := fmt.Sprintf("source_%s", src.Digest)
		atom := memory.MemoryAtom{
			ID:         sourceID,
			Kind:       memory.NodeResearchSource,
			Namespace:  ns,
			Provenance: prov,
			Confidence: src.Reliability,
			Data: memory.ResearchSourceAtom{
				SourceURI:       src.URI,
				Title:           src.Title,
				Authors:         src.Authors,
				PublicationDate: src.PublicationDate,
				Summary:         src.Summary,
				Digest:          src.Digest,
				FetchedAt:       bundle.FetchedAt,
			},
			Tags:      []string{"research", "standard", bundle.Domain},
			CreatedAt: now,
			UpdatedAt: now,
		}
		atoms = append(atoms, atom)
		sourceMap[src.URI] = sourceID
	}

	for _, claim := range bundle.Claims {
		claimID := fmt.Sprintf("claim_%s", claim.ID)
		ruleAtom := memory.MemoryAtom{
			ID:         claimID,
			Kind:       memory.NodeDesignRule,
			Namespace:  ns,
			Provenance: prov,
			Confidence: claim.Confidence,
			Data: memory.DesignRuleAtom{
				RuleID:         claim.ID,
				Axis:           bundle.Domain,
				Category:       claim.Subject,
				Title:          fmt.Sprintf("%s: %s", claim.Subject, claim.Object),
				Description:    claim.Context,
				HardConstraint: false,
				Weight:         claim.Confidence,
				Version:        "research_claim.v1",
			},
			Tags:      []string{"research_claim", claim.Subject, bundle.Domain},
			CreatedAt: now,
			UpdatedAt: now,
		}
		atoms = append(atoms, ruleAtom)

		if sourceAtomID, ok := sourceMap[claim.SourceURI]; ok {
			edges = append(edges, memory.MemoryEdge{
				FromID:     claimID,
				ToID:       sourceAtomID,
				Relation:   memory.RelDerivedFrom,
				Weight:     claim.Confidence,
				Provenance: prov,
				CreatedAt:  now,
			})
		}
	}

	return &memory.AdmissionBundle{SourceNamespace: ns, Atoms: atoms, Edges: edges}, nil
}

func (a *AdmissionAdapter) IngestBundle(ctx context.Context, bundle *ResearchBundle, targetScope memory.Namespace) error {
	if a.store == nil {
		return fmt.Errorf("admission adapter: memory store is nil")
	}
	admissionBundle, err := a.MapBundleToAdmission(bundle, targetScope)
	if err != nil {
		return err
	}
	return a.store.Commit(ctx, *admissionBundle)
}
