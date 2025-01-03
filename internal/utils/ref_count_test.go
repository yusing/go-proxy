package utils

import (
	"sync"
	"testing"
	"time"
)

func TestRefCounterAddSub(t *testing.T) {
	rc := NewRefCounter() // Count starts at 1

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rc.Add()
	}()

	go func() {
		defer wg.Done()
		rc.Sub()
		rc.Sub()
	}()

	wg.Wait()

	select {
	case <-rc.Zero():
		// Expected behavior
	case <-time.After(1 * time.Second):
		t.Fatal("Expected Zero channel to close, but it didn't")
	}
}

func TestRefCounterMultipleAddSub(t *testing.T) {
	rc := NewRefCounter()

	var wg sync.WaitGroup
	numAdds := 5
	numSubs := 5
	wg.Add(numAdds + numSubs)

	for range numAdds {
		go func() {
			defer wg.Done()
			rc.Add()
		}()
	}

	for range numSubs {
		go func() {
			defer wg.Done()
			rc.Sub()
			rc.Sub()
		}()
	}

	wg.Wait()

	select {
	case <-rc.Zero():
		// Expected behavior
	case <-time.After(1 * time.Second):
		t.Fatal("Expected Zero channel to close, but it didn't")
	}
}

func TestRefCounterOneInitially(t *testing.T) {
	rc := NewRefCounter()
	rc.Sub() // Bring count to zero

	select {
	case <-rc.Zero():
		// Expected behavior
	case <-time.After(1 * time.Second):
		t.Fatal("Expected Zero channel to close, but it didn't")
	}
}
