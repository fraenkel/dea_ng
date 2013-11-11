package health_check_test

import (
	. "dea/health_check"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
	"strconv"
	"time"
)

const HOST = "127.0.0.1"

type hcCallback struct {
	result chan bool
}

func newHCCallback() hcCallback {
	return hcCallback{make(chan bool)}
}

func (hcc hcCallback) Success() {
	hcc.result <- true
}
func (hcc hcCallback) Failure() {
	hcc.result <- false
}

func grabPort() string {
	listener, err := net.Listen("tcp", HOST+":0")
	Expect(err).To(BeNil())
	listener.Close()
	return listener.Addr().String()
}

var _ = Describe("PortOpen", func() {
	hostPort := grabPort()
	var port uint32

	BeforeEach(func() {
		_, p, _ := net.SplitHostPort(hostPort)
		u, err := strconv.ParseUint(p, 10, 32)
		Expect(err).To(BeNil())
		port = uint32(u)
	})

	It("should fail if no-one is listening on the port", func() {
		start := time.Now()
		ok := run_port_open(HOST, port, 100*time.Millisecond)
		elapsed := time.Now().Sub(start)

		Expect(ok).To(BeFalse())
		Expect(elapsed).To(BeNumerically(">", 90*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 120*time.Millisecond))
	})

	Context("starts a listener", func() {
		var listenDelay time.Duration
		var listener net.Listener

		BeforeEach(func() {
			listenDelay = 0
		})

		JustBeforeEach(func() {
			var err error
			if listenDelay == 0 {
				listener, err = net.Listen("tcp", hostPort)
				Expect(err).To(BeNil())
			} else {
				time.AfterFunc(listenDelay, func() {
					listener, err = net.Listen("tcp", hostPort)
					Expect(err).To(BeNil())
				})
			}
		})

		AfterEach(func() {
			if listener != nil {
				listener.Close()
			}
		})

		It("should succed if someone is listening on the port", func() {
			ok := run_port_open(HOST, port, 100*time.Millisecond)
			Expect(ok).To(BeTrue())
		})

		Context("delay listener", func() {
			BeforeEach(func() {
				listenDelay = 25 * time.Millisecond
			})

			It("should succed if someone starts listening on the port", func() {
				ok := run_port_open(HOST, port, 100*time.Millisecond)
				Expect(ok).To(BeTrue())
			})
		})
	})
})

func run_port_open(host string, port uint32, timeout time.Duration) bool {
	hcc := newHCCallback()

	portOpen := NewPortOpen(host, port, 2*time.Millisecond, hcc, timeout)
	defer portOpen.Destroy()

	return <-hcc.result
}
