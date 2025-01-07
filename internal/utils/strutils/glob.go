package strutils

import (
	"sync"

	"github.com/gobwas/glob"
)

var (
	globPatterns   = make(map[string]glob.Glob)
	globPatternsMu sync.Mutex
)

func GlobMatch(pattern string, s string) bool {
	if glob, ok := globPatterns[pattern]; ok {
		return glob.Match(s)
	}

	globPatternsMu.Lock()
	defer globPatternsMu.Unlock()

	glob, err := glob.Compile(pattern)
	if err != nil {
		return false
	}
	globPatterns[pattern] = glob
	return glob.Match(s)
}
