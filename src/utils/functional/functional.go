package functional

import "sync"

func ForEachKey[K comparable, V interface{}](obj map[K]V, do func(K)) {
	for k := range obj {
		do(k)
	}
}

func ForEachValue[K comparable, V interface{}](obj map[K]V, do func(V)) {
	for _, v := range obj {
		do(v)
	}
}

func ForEachKV[K comparable, V interface{}](obj map[K]V, do func(K, V)) {
	for k, v := range obj {
		do(k, v)
	}
}

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

func ParallelForEachKey[K comparable, V interface{}](obj map[K]V, do func(K)) {
	var wg sync.WaitGroup
	wg.Add(len(obj))
	for k := range obj {
		go func(k K) {
			do(k)
			wg.Done()
		}(k)
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

func ParallelForEachKV[K comparable, V interface{}](obj map[K]V, do func(K, V)) {
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
