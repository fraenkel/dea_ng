package utils

import (
	"sync"
)

func Parallel_promises(callbacks ...func() error) (result error) {
	wg := sync.WaitGroup{}
	wg.Add(len(callbacks) - 1)
	lock := sync.Mutex{}
	goCallbacks := callbacks[1:]
	for _, cb := range goCallbacks {
		curCB := cb
		go func() {
			defer wg.Done()
			if err := curCB(); err != nil {
				lock.Lock()
				defer lock.Unlock()
				if result != nil {
					result = err
				}
			}
		}()
	}
	err := callbacks[0]()
	lock.Lock()
	defer lock.Unlock()
	if result != nil {
		result = err
	}

	wg.Wait()
	return
}

func Sequence_promises(callbacks ...func() error) error {
	for _, cb := range callbacks {
		err := cb()
		if err != nil {
			return err
		}
	}
	return nil
}
