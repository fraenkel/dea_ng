package responders

import "time"

const default_advertise_interval time.Duration = 5 * time.Second

type Responder interface {
	Start()
	Stop()
}

type LocatorResponder interface {
	Advertise()
}
