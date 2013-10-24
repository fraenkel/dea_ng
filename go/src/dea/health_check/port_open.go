package health_check

import (
	"fmt"
	"net"
	"time"
)

type PortOpen struct {
	host     string
	port     uint32
	retry    time.Duration
	deferred HealthCheckCallback
	end_time time.Time
	done     bool
	conn     net.Conn
}

func NewPortOpen(host string, port uint32, retryInterval time.Duration,
	hc HealthCheckCallback, timeout time.Duration) PortOpen {

	portOpen := PortOpen{
		host:     host,
		port:     port,
		retry:    retryInterval,
		deferred: hc,
		end_time: time.Now().Add(timeout),
	}

	go portOpen.attempt_connect()
	return portOpen
}

func (p PortOpen) attempt_connect() {
	if !p.done {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", p.host, p.port), 5*time.Second)
		if err == nil {
			p.done = true
			p.conn = conn
			p.deferred.Success()
			p.Destroy()
		} else {
			if time.Now().Before(p.end_time) {
				time.AfterFunc(p.retry, p.attempt_connect)
			} else {
				p.done = true
				p.deferred.Failure()
				p.Destroy()
			}
		}
	}
}

func (p PortOpen) Destroy() {
	p.done = true
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
}
