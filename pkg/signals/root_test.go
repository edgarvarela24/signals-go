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
