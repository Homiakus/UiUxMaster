package design

import "testing"

func TestDefaultRubricHasUniquePositiveAxes(t *testing.T) {
	axes := DefaultRubric()
	if len(axes) == 0 {
		t.Fatal("default rubric must not be empty")
	}

	seen := make(map[string]struct{}, len(axes))
	for _, axis := range axes {
		if axis.ID == "" {
			t.Fatal("axis ID must not be empty")
		}
		if axis.Name == "" || axis.Description == "" {
			t.Fatalf("axis %q must have a name and description", axis.ID)
		}
		if axis.Weight <= 0 {
			t.Fatalf("axis %q must have positive weight, got %v", axis.ID, axis.Weight)
		}
		if _, ok := seen[axis.ID]; ok {
			t.Fatalf("duplicate axis ID %q", axis.ID)
		}
		seen[axis.ID] = struct{}{}
	}
}
