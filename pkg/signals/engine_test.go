package signals

import "testing"

func TestEngine_StartAndClose(t *testing.T) {
	// This test fails until you create the Start function
	eng := Start()
	if eng == nil {
		t.Fatal("Start() returned a nil Engine")
	}

	// This test ensures Close() exists and doesn't panic
	eng.Close()
}

func TestEngine_Scope(t *testing.T) {
	eng := Start()
	defer eng.Close()

	// Test the scope of the engine
	scope := eng.Scope()
	if scope == nil {
		t.Fatal("Scope() returned a nil Scope")
	}
}

func TestEngine_CloseDisposesScope(t *testing.T) {
	eng := Start()
	s := eng.Scope()

	// Close the engine, which should dispose the root scope.
	eng.Close()

	var executed bool
	s.Batch(func() {
		executed = true
	})

	if executed {
		t.Error("Batch function ran in a disposed scope, but it should not have.")
	}
}

func TestScope_Batch(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	var executed bool
	s.Batch(func() {
		executed = true
	})

	if !executed {
		t.Error("Batch function was not executed.")
	}
}

func TestEngine_CloseIsIdempotent(t *testing.T) {
	eng := Start()
	if err := eng.Close(); err != nil {
		t.Fatalf("first Close() returned an error: %v", err)
	}

	// Subsequent calls should not panic and should ideally return an error
	if err := eng.Close(); err == nil {
		t.Error("second Close() did not return an error, but it should have")
	}
}
