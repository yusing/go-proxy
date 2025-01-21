//go:build pprof

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

func initProfiling() {
	runtime.GOMAXPROCS(2)
	go func() {
		log.Println(http.ListenAndServe(":7777", nil))
	}()
}
