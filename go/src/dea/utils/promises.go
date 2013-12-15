package utils

import (
	"fmt"
	"runtime"
	"sync"
)

type Promise func() error

var promiseLogger = Logger("Promise", nil)

func Exec_promise(p Promise, callback func(error) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				err = r.(error)
			default:
				err = toError(r)
			}
		}

		if callback != nil {
			err = callback(err)
		}
	}()

	err = p()
	return
}

func Async_promise(p Promise, callback func(error) error) {
	go func() {
		err := Exec_promise(p, callback)
		if err != nil {
			promiseLogger.Errorf("Error occurred during promise: %s", err.Error())
		}

	}()
}

func Parallel_promises(promises ...Promise) (result error) {
	if len(promises) == 0 {
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
	wg.Add(len(promises) - 1)
	goCallbacks := promises[1:]
	for _, p := range goCallbacks {
		curP := p
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
			if err := curP(); err != nil {
				lock.Lock()
				defer lock.Unlock()
				if result == nil {
					result = err
				}
			}
		}()
	}
	err := promises[0]()

	wg.Wait()

	if result == nil {
		result = err
	}

	return
}

func Sequence_promises(promises ...Promise) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = toError(r)
		}
	}()

	for _, p := range promises {
		err = p()
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
