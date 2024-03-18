package main

import "sync"

func ParallelForEach[T interface{}](obj []T, do func(T)) {
	var wg sync.WaitGroup
	wg.Add(len(obj))
	for _, v := range obj {
		go func(v T) {
			do(v)
			wg.Done()
		}(v)
	}
	wg.Wait()
}

func ParallelForEachValue[K comparable, V interface{}](obj map[K]V, do func(V)) {
	var wg sync.WaitGroup
	wg.Add(len(obj))
	for _, v := range obj {
		go func(v V) {
			do(v)
			wg.Done()
		}(v)
	}
	wg.Wait()
}

func ParallelForEachKeyValue[K comparable, V interface{}](obj map[K]V, do func(K, V)) {
	var wg sync.WaitGroup
	wg.Add(len(obj))
	for k, v := range obj {
		go func(k K, v V) {
			do(k, v)
			wg.Done()
		}(k, v)
	}
	wg.Wait()
}