package visualdiff

// MetricsByEnvironment derives baseline lifecycle churn by the exact environment
// key that owned the baseline before each update. It is computed from durable
// audit history rather than a second mutable counter, so metrics cannot drift
// from the update records they summarize.
func (s *MemoryBaselineStore) MetricsByEnvironment() map[string]BaselineMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]BaselineMetrics)
	for id, ref := range s.baselines {
		creationKey := ref.EnvironmentKey
		if history := s.history[id]; len(history) > 0 {
			creationKey = history[0].OldEnvironmentKey
		}
		m := out[creationKey]
		m.Creates++
		out[creationKey] = m
	}
	for _, history := range s.history {
		for _, record := range history {
			m := out[record.OldEnvironmentKey]
			m.Updates++
			if record.OldEnvironmentKey != record.NewEnvironmentKey {
				m.EnvironmentChanges++
			}
			out[record.OldEnvironmentKey] = m
		}
	}
	return out
}
