package utils

import (
	"sync"
	"testing"
	"time"
)

func TestRefCounter_AddSub(t *testing.T) {
	rc := NewRefCounter()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rc.Add()
	}()

	go func() {
		defer wg.Done()
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

func TestRefCounter_MultipleAddSub(t *testing.T) {
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

func TestRefCounter_ZeroInitially(t *testing.T) {
	rc := NewRefCounter()
	rc.Sub() // Bring count to zero

	select {
	case <-rc.Zero():
		// Expected behavior
	case <-time.After(1 * time.Second):
		t.Fatal("Expected Zero channel to close, but it didn't")
	}
}
