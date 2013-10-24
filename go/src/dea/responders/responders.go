package responders

import "time"

const default_advertise_interval time.Duration = 5

type Responder interface {
	Start()
	Stop()
}

type LocatorResponder interface {
	Advertise()
}
