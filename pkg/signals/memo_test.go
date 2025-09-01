package signals

import "testing"

func TestMemo_ReturnsComputedValue(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)

	doubleCount := Memo(s, func() int {
		return count.Get() * 2
	})

	if doubleCount.Get() != 20 {
		t.Errorf("expected 20, got %d", doubleCount.Get())
	}
}

func TestMemo_IsLazy(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0

	doubleCount := Memo(s, func() int {
		runCount++
		return count.Get() * 2
	})

	if runCount != 0 {
		t.Fatalf("Expected memo to be lazy and not run on creation, ran %d times", runCount)
	}

	count.Set(20)

	if runCount != 0 {
		t.Fatalf("Expected memo to be lazy and not run on dependency change, ran %d times", runCount)
	}

	_ = doubleCount.Get()

	if runCount != 1 {
		t.Errorf("Expected memo to run once on first Get(), ran %d times", runCount)
	}
}

func TestMemo_CachesValue(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0

	doubleCount := Memo(s, func() int {
		runCount++
		return count.Get() * 2
	})

	// First read, should run the computation
	_ = doubleCount.Get()
	if runCount != 1 {
		t.Fatalf("Expected computation to run once on first read, ran %d times", runCount)
	}

	// Second read, should use cache
	_ = doubleCount.Get()
	if runCount != 1 {
		t.Errorf("Expected computation to be cached, but it ran again. Total runs: %d", runCount)
	}
}

func TestMemo_UpdatesWhenDependencyChanges(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	count := New(s, 10)
	runCount := 0
	doubleCount := Memo(s, func() int {
		runCount++
		return count.Get() * 2
	})

	// Initial read
	if val := doubleCount.Get(); val != 20 {
		t.Fatalf("Expected initial value to be 20, got %d", val)
	}
	if runCount != 1 {
		t.Fatalf("Expected computation to run once on first read, ran %d times", runCount)
	}

	// Change the dependency
	count.Set(30)

	// Read again, should re-compute
	if val := doubleCount.Get(); val != 60 {
		t.Errorf("Expected updated value to be 60, got %d", val)
	}
	if runCount != 2 {
		t.Errorf("Expected computation to run again after dependency change, ran %d times", runCount)
	}
}

func TestMemo_WorksWithMultipleDependencies(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	a := New(s, 10)
	b := New(s, 20)
	runCount := 0

	sum := Memo(s, func() int {
		runCount++
		return a.Get() + b.Get()
	})

	// Initial read
	if val := sum.Get(); val != 30 {
		t.Fatalf("Expected initial value to be 30, got %d", val)
	}
	if runCount != 1 {
		t.Fatalf("Expected computation to run once on first read, ran %d times", runCount)
	}

	// Change first dependency
	a.Set(15)
	if val := sum.Get(); val != 35 {
		t.Errorf("Expected updated value to be 35, got %d", val)
	}
	if runCount != 2 {
		t.Errorf("Expected computation to run again after first dependency change, ran %d times", runCount)
	}

	// Change second dependency
	b.Set(25)
	if val := sum.Get(); val != 40 {
		t.Errorf("Expected updated value to be 40, got %d", val)
	}
	if runCount != 3 {
		t.Errorf("Expected computation to run again after second dependency change, ran %d times", runCount)
	}
}

func TestMemo_WorksWhenNested(t *testing.T) {
	eng := Start()
	defer eng.Close()
	s := eng.Scope()

	a := New(s, 10)
	bRunCount := 0
	cRunCount := 0

	// b depends on a
	b := Memo(s, func() int {
		bRunCount++
		return a.Get() * 2
	})

	// c depends on b
	c := Memo(s, func() int {
		cRunCount++
		return b.Get() + 1
	})

	// Initial read
	if val := c.Get(); val != 21 {
		t.Fatalf("Expected initial value to be 21, got %d", val)
	}
	if bRunCount != 1 || cRunCount != 1 {
		t.Fatalf("Expected initial run counts to be 1, got b=%d, c=%d", bRunCount, cRunCount)
	}

	// Change the root dependency
	a.Set(20)

	// Read again, should re-compute the entire chain
	if val := c.Get(); val != 41 {
		t.Errorf("Expected updated value to be 41, got %d", val)
	}
	if bRunCount != 2 || cRunCount != 2 {
		t.Errorf("Expected run counts to be 2 after update, got b=%d, c=%d", bRunCount, cRunCount)
	}
}
