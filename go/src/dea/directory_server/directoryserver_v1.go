package directory_server

import (
	"dea/starting"
	"encoding/base64"
	"github.com/nu7hatch/gouuid"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type DirectoryServerV1 struct {
	host             string
	Port             uint16
	Uri              string
	Credentials      []string
	instanceRegistry *starting.InstanceRegistry
	uuid             *uuid.UUID
}

func NewDirectoryServerV1(localIp string, port uint16, instanceRegistry *starting.InstanceRegistry) (*DirectoryServerV1, error) {
	address := localIp + ":" + strconv.Itoa(int(port))
	dirServer := &DirectoryServerV1{
		host:             localIp,
		Port:             port,
		Uri:              "http://" + address + "/instances",
		Credentials:      []string{},
		instanceRegistry: instanceRegistry,
	}

	u, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	dirServer.Credentials[0] = u.String()

	u, err = uuid.NewV4()
	if err != nil {
		return nil, err
	}
	dirServer.Credentials[1] = u.String()

	dirServer.uuid, err = uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return dirServer, nil
}

func (d *DirectoryServerV1) Start() error {
	address := d.host + ":" + strconv.Itoa(int(d.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	err = http.Serve(listener, d)
	if err != nil {
		return err
	}

	return nil
}

var instancesPath, _ = url.Parse("/instances")

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

	if r.Method == "GET" && r.URL == instancesPath {
		NewDirectory(dirServer.instanceRegistry).Handle(w)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
