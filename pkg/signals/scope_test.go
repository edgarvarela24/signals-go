package signals

import "testing"

func TestSignal_New(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)

	if count == nil {
		t.Fatal("Expected non-nil Signal")
	}
}
