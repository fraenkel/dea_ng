package utils

import (
	"fmt"
	"runtime"
	"sync"
)

func Parallel_promises(callbacks ...func() error) (result error) {
	if len(callbacks) == 0 {
		return nil
	}

	lock := sync.Mutex{}
	defer func() {
		if r := recover(); r != nil {
			lock.Lock()
			defer lock.Unlock()

			result = toError(r)
			return
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(len(callbacks) - 1)
	goCallbacks := callbacks[1:]
	for _, cb := range goCallbacks {
		curCB := cb
		go func() {
			defer func() {
				if r := recover(); r != nil {
					lock.Lock()
					defer lock.Unlock()

					result = toError(r)
					return
				}
			}()

			defer wg.Done()
			if err := curCB(); err != nil {
				lock.Lock()
				defer lock.Unlock()
				if result == nil {
					result = err
				}
			}
		}()
	}
	err := callbacks[0]()

	wg.Wait()

	if result == nil {
		result = err
	}

	return
}

func Sequence_promises(callbacks ...func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = toError(r)
		}
	}()

	for _, cb := range callbacks {
		err = cb()
		if err != nil {
			return
		}
	}

	return
}

func toError(r interface{}) error {
	if r == nil {
		return nil
	}
	stack := make([]byte, 1024)
	n := runtime.Stack(stack, false)
	return fmt.Errorf("Panic: %v\n Stack:\n%s", r, string(stack[:n]))
}
