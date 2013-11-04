package directory_server

import (
	"encoding/base64"
	"github.com/nu7hatch/gouuid"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DirectoryServerV1 struct {
	host        string
	Port        uint16
	Uri         string
	Credentials []string
	handler     http.Handler
	listener    net.Listener
	uuid        *uuid.UUID
}

func NewDirectoryServerV1(localIp string, port uint16, handler http.Handler) (*DirectoryServerV1, error) {
	dirServer := &DirectoryServerV1{
		host:    localIp,
		Port:    port,
		handler: handler,
	}

	creds := make([]string, 2)
	u, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	creds[0] = u.String()

	u, err = uuid.NewV4()
	if err != nil {
		return nil, err
	}
	creds[1] = u.String()

	dirServer.uuid, err = uuid.NewV4()
	if err != nil {
		return nil, err
	}

	dirServer.Credentials = creds
	return dirServer, nil
}

func (d *DirectoryServerV1) Start() error {
	address := d.host + ":" + strconv.Itoa(int(d.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	d.listener = listener
	listenAddr := listener.Addr().String()
	d.Uri = "http://" + listenAddr + "/instances"
	if d.Port == 0 {
		_, listenerPort, err := net.SplitHostPort(listenAddr)
		if err == nil {
			port, err := strconv.ParseUint(listenerPort, 10, 16)
			if err == nil {
				d.Port = uint16(port)
			}
		}
	}

	server := http.Server{
		Handler:      d,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go server.Serve(d.listener)

	return nil
}

func (d *DirectoryServerV1) Stop() {
	d.listener.Close()
}

const instancesPath = "/instances/"

func (dirServer *DirectoryServerV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		unauthorized(w)
		return
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		unauthorized(w)
		return
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		unauthorized(w)
		return
	}

	if len(pair) != len(dirServer.Credentials) {
		unauthorized(w)
		return
	}

	for i, v := range dirServer.Credentials {
		if pair[i] != v {
			unauthorized(w)
			return
		}
	}

	if r.Method == "GET" && r.URL.String() == instancesPath {
		dirServer.handler.ServeHTTP(w, r)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
