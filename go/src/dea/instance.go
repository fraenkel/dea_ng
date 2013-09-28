package dea

import (
	"fmt"
	"github.com/nu7hatch/gouuid"
	"strconv"
	"strings"
)

type State string

const (
	BORN     State = "BORN"
	STARTING State = "STARTING"
	RUNNING  State = "RUNNING"
	STOPPING State = "STOPPING"
	STOPPED  State = "STOPPED"
	CRASHED  State = "CRASHED"
	DELETED  State = "DELETED"
	RESUMING State = "RESUMING"
)

type Instance struct {
	attributes      map[string]interface{}
	state           State
	exitStatus      int
	exitDescription string
}

func NewInstance(raw_attributes map[string]interface{}) *Instance {
	attributes := translate_attributes(raw_attributes)

	if _, exists := attributes["application_uris"]; !exists {
		attributes["application_uris"] = []string{}
	}

	// Generate unique ID
	if _, exists := attributes["instance_id"]; !exists {
		attributes["instance_id"] = UUID()
	}

	// Contatenate 2 UUIDs to generate a 32 chars long private_instance_id
	if _, exists := attributes["private_instance_id"]; !exists {
		attributes["private_instance_id"] = UUID() + UUID()
	}

	instance := &Instance{}
	instance.attributes = attributes
	instance.state = BORN

	instance.exitStatus = -1
	instance.exitDescription = ""
	return instance
}

func (i *Instance) CCPartition() string {
	return i.attributes["cc_partition"].(string)
}

func (i *Instance) InstanceId() string {
	return i.attributes["instance_id"].(string)
}

func (i *Instance) InstanceIndex() int {
	return i.attributes["instance_index"].(int)
}

func (i *Instance) ApplicationId() string {
	return i.attributes["application_id"].(string)
}
func (i *Instance) ApplicationVersion() string {
	return i.attributes["application_version"].(string)
}
func (i *Instance) ApplicationName() string {
	return i.attributes["application_name"].(string)
}
func (i *Instance) ApplicationUris() []string {
	return i.attributes["application_uris"].([]string)
}
func (i *Instance) DropletSha1() string {
	return i.attributes["droplet_sha1"].(string)
}
func (i *Instance) DropletUri() string {
	return i.attributes["droplet_uri"].(string)
}
func (i *Instance) StartCommand() string {
	cmd, exists := i.attributes["start_command"]
	if exists {
		return cmd.(string)
	}
	return ""
}
func (i *Instance) Limits() map[string]int {
	return i.attributes["limits"].(map[string]int)
}
func (i *Instance) Environment() map[string]string {
	return i.attributes["environment"].(map[string]string)
}
func (i *Instance) Services() []map[string]interface{} {
	return i.attributes["services"].([]map[string]interface{})
}

func (i *Instance) PrivateInstanceId() string {
	return i.attributes["private_instance_id"].(string)
}

func (i *Instance) MemoryLimit() int {
	return i.Limits()["mem"] * 1024 * 1024
}

func (i *Instance) DiskLimit() int {
	return i.Limits()["disk"] * 1024 * 1024
}

func (i *Instance) FileDescriptorLimit() int {
	return i.Limits()["fds"]
}

func UUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x%x%x%x%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func translate_attributes(raw_attributes map[string]interface{}) map[string]interface{} {
	attributes := make(map[string]interface{})
	for k, v := range raw_attributes {
		attributes[k] = v
	}

	transfer_attribute_with_existence_check(attributes, "instance_index", "index")
	transfer_attribute_with_existence_check(attributes, "application_version", "version")
	transfer_attribute_with_existence_check(attributes, "application_name", "name")
	transfer_attribute_with_existence_check(attributes, "application_uris", "uris")

	if value, exists := attributes["droplet"]; exists {
		attributes["application_id"] = strconv.Itoa(value.(int))
	}
	delete(attributes, "droplet")

	transfer_attribute_with_existence_check(attributes, "", "")
	transfer_attribute(attributes, "droplet_sha1", "sha1")
	transfer_attribute(attributes, "droplet_uri", "executableUri")

	// Translate environment to dictionary (it is passed as Array with VAR = VAL)
	envIntf, exists := attributes["env"]
	if exists {
		delete(attributes, "env")

		if _, exists := attributes["environment"]; !exists && envIntf != nil {
			env := envIntf.([]string)
			env_hash := make(map[string]string)
			attributes["environment"] = env_hash
			for _, e := range env {
				pair := strings.SplitN(string(e), "=", 2)
				if len(pair) == 1 {
					pair = append(pair, "")
				}
				env_hash[pair[0]] = pair[1]
			}
		}
	}

	return attributes
}

func transfer_attribute(attributes map[string]interface{}, new_key string, old_key string) {
	value := attributes[old_key]
	delete(attributes, old_key)
	attributes[new_key] = value
}

func transfer_attribute_with_existence_check(attributes map[string]interface{}, new_key string, old_key string) {
	value, exists := attributes[old_key]
	if exists {
		attributes[new_key] = value
		delete(attributes, old_key)
	}
}
