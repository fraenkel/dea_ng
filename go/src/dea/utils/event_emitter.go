package utils

type EventEmitter struct {
	listeners map[interface{}][]func()
}

func (e EventEmitter) On(event interface{}, callback func()) {
	cbs := e.listeners[event]
	if cbs == nil {
		cbs = make([]func(), 0, 1)
	}
	e.listeners[event] = append(cbs, callback)
}

func (e EventEmitter) Emit(event interface{}) {
	if cbs := e.listeners[event]; cbs != nil {
		for _, cb := range cbs {
			cb()
		}
	}
}
