package utils

// empty struct that implements Locker interface
// for hinting that no copy should be performed.
type NoCopy struct{}

func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}
