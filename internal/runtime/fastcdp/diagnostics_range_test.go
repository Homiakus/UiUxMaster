package fastcdp

import "testing"

func TestDiagnosticsSnapshotRangeClosesCycleWithoutGap(t *testing.T) {
	o := &DiagnosticsObserver{capacity: 8, stop: make(chan struct{})}
	start := o.Mark()
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "cycle-1"})
	through := o.Mark()
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "cycle-2"})

	first := o.SnapshotRange(start, through)
	if !first.Complete || first.Through.Sequence != through.Sequence {
		t.Fatalf("first snapshot = %#v", first)
	}
	if len(first.Events) != 1 || first.Events[0].Message != "cycle-1" {
		t.Fatalf("first events = %#v", first.Events)
	}

	second := o.SnapshotRange(first.Through, o.Mark())
	if !second.Complete || len(second.Events) != 1 || second.Events[0].Message != "cycle-2" {
		t.Fatalf("second snapshot = %#v", second)
	}
}

func TestDiagnosticsSnapshotRangeRejectsReverseWatermark(t *testing.T) {
	o := &DiagnosticsObserver{capacity: 4, stop: make(chan struct{})}
	result := o.SnapshotRange(DiagnosticMark{Sequence: 3}, DiagnosticMark{Sequence: 2})
	if result.Complete || len(result.DroppedMethods) != 1 || result.DroppedMethods[0] != "observer.invalid_range" {
		t.Fatalf("result = %#v", result)
	}
}
