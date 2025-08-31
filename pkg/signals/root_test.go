package signals

import "testing"

// TestRootMinimal ensures the stub Root executes the provided function.
func TestRoot_NoOp(t *testing.T) {
	ran := false
	dispose := Root(func(s Scope) {
		ran = true
	})

	if !ran {
		t.Error("root function was not executed")
	}
	// dispose should be callable (currently no observable effect)
	dispose()
}

func TestRoot_DisposePreventsFutureWork(t *testing.T) {
	var s Scope
	var executed bool

	dispose := Root(func(scope Scope) {
		s = scope
	})

	// Dispose immediately
	dispose()

	s.Batch(func() {
		executed = true
	})

	if executed {
		t.Error("Batch function ran in a disposed scope, but it should not have.")
	}
}
