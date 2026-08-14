package syncs

import "sync"

func OnceValue[T any](f func() T) func() T {
	var once sync.Once
	var v T
	return func() T {
		once.Do(func() { v = f() })
		return v
	}
}
