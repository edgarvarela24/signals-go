package signals

import "testing"

func TestSignal_GetReturnsInitialValue(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)

	if val := count.Get(); val != 10 {
		t.Errorf("Expected initial value to be 10, got %v", val)
	}
}

func TestSignal_SetUpdatesValue(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)

	count.Set(20)
	if val := count.Get(); val != 20 {
		t.Errorf("Expected updated value to be 20, got %v", val)
	}
}

func TestSignal_UpdateMutatesValue(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)

	count.Update(func(val *int) {
		*val = *val * 3
	})
	if val := count.Get(); val != 30 {
		t.Errorf("Expected updated value to be 30, got %v", val)
	}
}
