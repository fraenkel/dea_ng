package dea

import (
	"dea/config"
	"dea/starting"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/go_cfmessagebus"
	"strconv"
)

var rcLogger = utils.Logger("RouterClient", nil)

type RouterClient struct {
	config   *config.Config
	mbus     cfmessagebus.MessageBus
	uuid     string
	local_ip string
}

func NewRouterClient(config *config.Config, nats *Nats, uuid, local_ip string) RouterClient {
	return RouterClient{config, nats.MessageBus, uuid, local_ip}
}

func (r *RouterClient) RegisterInstance(i *starting.Instance, opts map[string]interface{}) error {
	req := r.generate_instance_request(i, opts)
	return r.publish("router.register", req)
}

// Same format is used for both registration and unregistration
func (r *RouterClient) generate_instance_request(i *starting.Instance, opts map[string]interface{}) map[string]interface{} {
	uris := opts["uris"].([]string)
	if uris == nil {
		uris = i.ApplicationUris()
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

func (r *RouterClient) Greet(callback func(response []byte)) error {
	err := r.mbus.Request("router.greet", []byte{'{', '}'}, callback)
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

	return r.mbus.Publish(subject, bytes)
}

/*

module Dea
  class RouterClient
    attr_reader :bootstrap

    def initialize(bootstrap)
      @bootstrap = bootstrap
    end


    def register_instance(instance, opts = {})
      req = generate_instance_request(instance, opts)
      bootstrap.nats.publish("router.register", req)
    end

    def unregister_instance(instance, opts = {})
      req = generate_instance_request(instance, opts)
      bootstrap.nats.publish("router.unregister", req)
    end

    def greet(&blk)
      bootstrap.nats.request("router.greet", &blk)
    end

    private

    # Same format is used for both registration and unregistration
    def generate_instance_request(instance, opts = {})
      { "dea"  => bootstrap.uuid,
        "app"  => instance.application_id,
        "uris" => opts[:uris] || instance.application_uris,
        "host" => bootstrap.local_ip,
        "port" => instance.instance_host_port,
        "tags" => { "component" => "dea-#{bootstrap.config["index"]}" },
        "private_instance_id" => instance.private_instance_id,
      }
    end

    # Same format is used for both registration and unregistration
    def generate_directory_server_request(host, port, uri)
      { "host" => host,
        "port" => port,
        "uris" => [uri],
        "tags" => { "component" => "directory-server-#{bootstrap.config["index"]}" },
      }
    end
  end
end
*/
