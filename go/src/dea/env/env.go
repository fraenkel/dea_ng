package env

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var serviceKeys = []string{"name", "label", "tags", "plan", "plan_option", "credentials"}

type Env struct {
	envStrategy EnvStrategy
	vcapApp     *map[string]interface{}
	vcapSrvcs   *map[string][]map[string]interface{}
}

type Message interface {
	Name() string
	Version() string
	Uris() []string
	Limits() map[string]uint64
	MemoryLimit() uint64
	Services() []map[string]interface{}
	Env() []string
}

type EnvStrategy interface {
	ExportedSystemEnvironmentVariables() [][]string
	VcapApplication() map[string]interface{}
	Message() Message
}

func NewEnv(strategy EnvStrategy) *Env {
	env := &Env{envStrategy: strategy}
	return env
}

func (e Env) ExportedSystemEnvironmentVariables() (string, error) {
	vcapApp, err := json.Marshal(e.vcapApplication())
	if err != nil {
		return "", err
	}
	vcapServices, err := json.Marshal(e.vcapServices())
	if err != nil {
		return "", err
	}

	envList := make([][]string, 0, 4)
	envList[0] = []string{"VCAP_APPLICATION", string(vcapApp)}
	envList[1] = []string{"VCAP_SERVICES", string(vcapServices)}

	memlimit := e.envStrategy.Message().MemoryLimit()
	envList[2] = []string{"MEMORY_LIMIT", strconv.FormatUint(memlimit, 10) + "m"}

	services := e.envStrategy.Message().Services()
	if len(services) > 0 {
		dbGen := NewDatabaseUriGenerator(services)
		dbUri, err := dbGen.DatabaseUri()
		if err != nil {
			return "", err
		}
		if dbUri != nil {
			envList = append(envList, []string{"DATABASE_URL", dbUri.String()})
		}
	}

	return to_export(envList, e.envStrategy.ExportedSystemEnvironmentVariables()), nil
}

func (e Env) ExportedUserEnvironmentVariables() string {
	return to_export(translate_env(e.envStrategy.Message().Env()))
}

func (e Env) ExportedEnvironmentVariables() (string, error) {
	sysEnv, err := e.ExportedSystemEnvironmentVariables()
	if err != nil {
		return "", err
	}
	return sysEnv + e.ExportedUserEnvironmentVariables(), nil
}

func (e Env) vcapApplication() map[string]interface{} {
	if e.vcapApplication == nil {
		msg := e.envStrategy.Message()
		vcapApp := e.envStrategy.VcapApplication()
		vcapApp["limits"] = msg.Limits()
		vcapApp["application_version"] = msg.Version()
		vcapApp["application_name"] = msg.Name()
		vcapApp["application_uris"] = msg.Uris()
		// Translate keys for backwards compatibility
		vcapApp["version"] = vcapApp["application_version"]
		vcapApp["name"] = vcapApp["application_name"]
		vcapApp["uris"] = vcapApp["application_uris"]
		vcapApp["users"] = vcapApp["application_users"]
		e.vcapApp = &vcapApp
	}
	return *e.vcapApp
}

func (e Env) vcapServices() map[string][]map[string]interface{} {
	if e.vcapSrvcs == nil {
		services := make(map[string][]map[string]interface{})
		servicesData := e.envStrategy.Message().Services()

		for _, serviceInfo := range servicesData {
			serviceMap := make(map[string]interface{})
			for _, k := range serviceKeys {
				if v, exists := serviceInfo[k]; exists {
					serviceMap[k] = v
				}
			}

			label := serviceMap["label"].(string)
			serviceList, exists := services[label]
			if !exists {
				serviceList = make([]map[string]interface{}, 0, 1)
			}

			services[label] = append(serviceList, serviceMap)
		}
		e.vcapSrvcs = &services
	}

	return *e.vcapSrvcs
}

func translate_env(env []string) [][]string {
	envs := make([][]string, len(env))

	for _, e := range env {
		pair := strings.SplitN(string(e), "=", 2)
		if len(pair) == 1 {
			pair = append(pair, "")
		}
		envs = append(envs, pair)
	}
	return envs
}

func to_export(multienvs ...[][]string) string {
	count := 0
	for _, envs := range multienvs {
		count += len(envs)
	}
	exports := make([]string, 0, count)
	for _, envs := range multienvs {
		for _, env := range envs {
			escaped := fmt.Sprintf("export %s=\"%s\";\n", env[0], strings.Replace(env[1], "\"", "\\\"", -1))
			exports = append(exports, escaped)
		}
	}

	return strings.Join(exports, "")
}

/*# coding: UTF-8

VCAP_SERVICES=
{
  cleardb-n/a: [
    {
      name: "cleardb-1",
      label: "cleardb-n/a",
      plan: "spark",
      credentials: {
        name: "ad_c6f4446532610ab",
        hostname: "us-cdbr-east-03.cleardb.com",
        port: "3306",
        username: "b5d435f40dd2b2",
        password: "ebfc00ac",
        uri: "mysql://b5d435f40dd2b2:ebfc00ac@us-cdbr-east-03.cleardb.com:3306/ad_c6f4446532610ab",
        jdbcUrl: "jdbc:mysql://b5d435f40dd2b2:ebfc00ac@us-cdbr-east-03.cleardb.com:3306/ad_c6f4446532610ab"
      }
    }
  ],
  cloudamqp-n/a: [
    {
      name: "cloudamqp-6",
      label: "cloudamqp-n/a",
      plan: "lemur",
      credentials: {
        uri: "amqp://ksvyjmiv:IwN6dCdZmeQD4O0ZPKpu1YOaLx1he8wo@lemur.cloudamqp.com/ksvyjmiv"
      }
    }
  ],


*/
