package utils

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var httpTimeout = HttpTimeouts{5 * time.Second, 5 * time.Second}

type HttpTimeouts struct {
	Connect time.Duration
	Read    time.Duration
}

func GetHttpTimeouts() HttpTimeouts {
	return httpTimeout
}

func SetHttpTimeouts(timeouts HttpTimeouts) {
	httpTimeout = timeouts
}

func newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, httpTimeout.Connect)
		},
		ResponseHeaderTimeout: httpTimeout.Read,
	}
	return &http.Client{
		Transport: tr,
	}
}
