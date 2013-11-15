package router_client

import (
	"dea"
	"dea/config"
	"dea/starting"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/yagnats"
	"strconv"
)

var rcLogger = utils.Logger("RouterClient", nil)

type RouterClient struct {
	config   *config.Config
	nats     *dea.Nats
	uuid     string
	local_ip string
}

func NewRouterClient(config *config.Config, nats *dea.Nats, uuid, local_ip string) RouterClient {
	return RouterClient{config, nats, uuid, local_ip}
}

func (r *RouterClient) RegisterInstance(i *starting.Instance, opts map[string]interface{}) error {
	req := r.generate_instance_request(i, opts)
	return r.publish("router.register", req)
}

// Same format is used for both registration and unregistration
func (r *RouterClient) generate_instance_request(i *starting.Instance, opts map[string]interface{}) map[string]interface{} {
	uris := i.ApplicationUris()

	if opts != nil {
		if u, ok := opts["uris"].([]string); ok {
			uris = u
		}
	}

	rsp := map[string]interface{}{
		"dea":                 r.uuid,
		"app":                 i.ApplicationId(),
		"uris":                uris,
		"host":                r.local_ip,
		"port":                i.HostPort(),
		"tags":                map[string]string{"component": "dea-" + strconv.FormatUint(uint64(r.config.Index), 10)},
		"private_instance_id": i.PrivateInstanceId(),
	}

	return rsp
}

func (r *RouterClient) UnregisterInstance(i *starting.Instance, opts map[string]interface{}) error {
	req := r.generate_instance_request(i, opts)
	return r.publish("router.unregister", req)
}

func (r *RouterClient) Greet(callback yagnats.Callback) error {
	_, err := r.nats.Request("router.greet", []byte("{}"), callback)
	if err != nil {
		rcLogger.Errorf("greet error: %s", err.Error())
	}
	return err
}

func (r *RouterClient) Register_directory_server(host string, port uint16, uri string) error {
	req := r.generate_directory_server_request(host, port, uri)
	return r.publish("router.register", req)
}

func (r *RouterClient) Unregister_directory_server(host string, port uint16, uri string) error {
	req := r.generate_directory_server_request(host, port, uri)
	return r.publish("router.unregister", req)
}

// Same format is used for both registration and unregistration
func (r *RouterClient) generate_directory_server_request(host string, port uint16, uri string) map[string]interface{} {
	return map[string]interface{}{
		"host": host,
		"port": port,
		"uris": []string{uri},
		"tags": map[string]string{"component": "directory-server-" + strconv.FormatUint(uint64(r.config.Index), 10)},
	}
}

func (r *RouterClient) publish(subject string, message interface{}) error {
	bytes, err := json.Marshal(message)
	if err != nil {
		rcLogger.Errorf("%s error: %s", subject, err.Error())
		return err
	}

	return r.nats.NatsClient.Publish(subject, bytes)
}
