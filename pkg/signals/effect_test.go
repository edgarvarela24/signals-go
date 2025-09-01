package signals

import "testing"

func TestEffect_RunsOnSignalChanges(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0
	Effect(s, func() {
		_ = count.Get() // Establish a dependency on `count`
		runCount++
	})

	if runCount != 1 {
		t.Errorf("Expected effect to run once on creation, ran %d times", runCount)
	}

	count.Set(20)
	if runCount != 2 {
		t.Errorf("Expected effect to run again on signal change, ran %d times", runCount)
	}
}

func TestEffect_OnlyRunsOnDependentSignalChanges(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	a := New(s, 10)
	b := New(s, 20)
	aRan := 0

	Effect(s, func() {
		_ = a.Get()
		aRan++
	})

	b.Set(30)

	if aRan != 1 {
		t.Errorf("Expected aRan to be 1, got %d", aRan)
	}
}

func TestEffect_BatchedChangesRunOnce(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0
	Effect(s, func() {
		_ = count.Get() // Establish a dependency on `count`
		runCount++
	})

	if runCount != 1 {
		t.Errorf("Expected effect to run once on creation, ran %d times", runCount)
	}

	s.Batch(func() {
		count.Set(20)
		count.Set(30)
		count.Set(40)
	})

	if runCount != 2 {
		t.Errorf("Expected effect to run only once for batched changes, ran %d times", runCount)
	}
}

func TestEffect_UntrackPreventsDependencies(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	a := New(s, 10)
	b := New(s, 20)
	runCount := 0

	Effect(s, func() {
		_ = a.Get() // Dependency on `a`
		Untrack(s, func() {
			_ = b.Get() // No dependency on `b`
		})
		runCount++
	})

	if runCount != 1 {
		t.Fatalf("Expected effect to run once on creation, ran %d times", runCount)
	}

	b.Set(30)
	if runCount != 1 {
		t.Errorf("Expected effect not to run on untracked dependency change, ran %d times", runCount)
	}

	a.Set(15)
	if runCount != 2 {
		t.Errorf("Expected effect to run on tracked dependency change, ran %d times", runCount)
	}
}

func TestEffect_OnCleanupIsCalled(t *testing.T) {
	eng := Start()
	s := eng.Scope()

	count := 0
	Effect(s, func() {
		OnCleanup(s, func() {
			count++
		})
	})

	eng.Close()

	if count != 1 {
		t.Errorf("Expected OnCleanup to be called once, got %d", count)
	}
}

func TestEffect_ReturnedCleanupStopsEffect(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0

	// Effect now returns a cleanup function
	stopEffect := Effect(s, func() {
		_ = count.Get()
		runCount++
	})

	if runCount != 1 {
		t.Fatalf("Expected effect to run once on creation, ran %d times", runCount)
	}

	// Stop the effect manually
	stopEffect()

	// Change the dependency
	count.Set(20)

	// The effect should NOT have run again
	if runCount != 1 {
		t.Errorf("Expected effect to be stopped, but it ran again. Total runs: %d", runCount)
	}
}
