package fastcdp

import "testing"

func TestNearestProjectedElementParentSkipsNonLayoutWrapper(t *testing.T) {
	parents := []int{-1, 0, 1, 2}
	projected := map[int]struct{}{0: {}, 2: {}, 3: {}}

	if got := nearestProjectedElementParent(parents, 2, projected); got != 0 {
		t.Fatalf("parent of node 2 = %d, want nearest projected ancestor 0", got)
	}
	if got := nearestProjectedElementParent(parents, 3, projected); got != 2 {
		t.Fatalf("parent of node 3 = %d, want direct projected ancestor 2", got)
	}
	if got := nearestProjectedElementParent(parents, 0, projected); got != -1 {
		t.Fatalf("root parent = %d, want -1", got)
	}
}

func TestNearestProjectedElementParentStopsOnCycle(t *testing.T) {
	parents := []int{1, 0}
	if got := nearestProjectedElementParent(parents, 0, map[int]struct{}{}); got != -1 {
		t.Fatalf("cycle parent = %d, want -1", got)
	}
}
