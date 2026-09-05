package evidence

// RenderFreshness attests which committed render revision was captured. Epoch
// remains monotonic sequencing data; ExpectedRevision/ObservedRevision bind that
// sequence to the source/build/change identity requested by the validation run.
type RenderFreshness struct {
	Epoch            uint64 `json:"epoch"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	ObservedRevision string `json:"observed_revision,omitempty"`
}

// Matches reports whether revision-bound evidence satisfies its declared
// expectation. Revisionless legacy packets remain outside this contract.
func (f RenderFreshness) Matches() bool {
	return f.ExpectedRevision == "" || f.ExpectedRevision == f.ObservedRevision
}
