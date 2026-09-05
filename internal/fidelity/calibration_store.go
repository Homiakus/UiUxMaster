package fidelity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const CalibrationSnapshotSchemaVersion = 1

func IsCalibrationMissing(err error) bool             { return errors.Is(err, ErrCalibrationMissing) }
func IsCalibrationEnvironmentMismatch(err error) bool { return errors.Is(err, ErrCalibrationEnvironmentMismatch) }
func IsCalibrationExpired(err error) bool             { return errors.Is(err, ErrCalibrationExpired) }
func IsCalibrationCoverage(err error) bool            { return errors.Is(err, ErrCalibrationCoverage) }
func IsCalibrationQuality(err error) bool             { return errors.Is(err, ErrCalibrationQuality) }

// CalibrationSnapshot is the durable parity-corpus envelope. Every record
// already contains its canonical environment key, corpus digest and artifact
// reference, so persisted calibration can always be traced to the exact runtime
// pair that produced it.
type CalibrationSnapshot struct {
	SchemaVersion int                 `json:"schema_version"`
	Records       []CalibrationRecord `json:"records"`
}

func (r *CalibrationRegistry) Snapshot() CalibrationSnapshot {
	snapshot := CalibrationSnapshot{SchemaVersion: CalibrationSnapshotSchemaVersion}
	if r == nil {
		return snapshot
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, byTier := range r.records {
		for _, records := range byTier {
			snapshot.Records = append(snapshot.Records, records...)
		}
	}
	sort.Slice(snapshot.Records, func(i, j int) bool {
		if snapshot.Records[i].Class != snapshot.Records[j].Class {
			return snapshot.Records[i].Class < snapshot.Records[j].Class
		}
		if snapshot.Records[i].Tier != snapshot.Records[j].Tier {
			return snapshot.Records[i].Tier < snapshot.Records[j].Tier
		}
		return snapshot.Records[i].CreatedAt.Before(snapshot.Records[j].CreatedAt)
	})
	return snapshot
}

func RestoreCalibrationRegistry(snapshot CalibrationSnapshot) (*CalibrationRegistry, error) {
	if snapshot.SchemaVersion != CalibrationSnapshotSchemaVersion {
		return nil, fmt.Errorf("fidelity: unsupported calibration snapshot schema %d", snapshot.SchemaVersion)
	}
	registry := NewCalibrationRegistry()
	for _, record := range snapshot.Records {
		if err := registry.Put(record); err != nil {
			return nil, fmt.Errorf("fidelity: restore calibration record: %w", err)
		}
	}
	return registry, nil
}

// SaveFile writes calibration state atomically. The temporary file is fsynced
// before rename so an interrupted write cannot replace a valid snapshot with a
// partially written JSON document.
func (r *CalibrationRegistry) SaveFile(path string) error {
	if path == "" {
		return fmt.Errorf("fidelity: calibration snapshot path is required")
	}
	data, err := json.MarshalIndent(r.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".calibration-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func LoadCalibrationRegistry(path string) (*CalibrationRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snapshot CalibrationSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("fidelity: decode calibration snapshot: %w", err)
	}
	return RestoreCalibrationRegistry(snapshot)
}
