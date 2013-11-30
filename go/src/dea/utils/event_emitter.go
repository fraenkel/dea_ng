package utils

type Event interface {
	Name() string
}

type EventEmitter struct {
	listeners map[interface{}][]func()
}

func NewEventEmitter() EventEmitter {
	return EventEmitter{listeners: make(map[interface{}][]func())}
}

func (e EventEmitter) On(event interface{}, callback func()) {
	if n, ok := event.(Event); ok {
		event = n
	}

	cbs := e.listeners[event]
	if cbs == nil {
		cbs = make([]func(), 0, 1)
	}
	e.listeners[event] = append(cbs, callback)
}

func (e EventEmitter) Emit(event interface{}) {
	if n, ok := event.(Event); ok {
		event = n
	}

	if cbs := e.listeners[event]; cbs != nil {
		for _, cb := range cbs {
			cb()
		}
	}
}
