package env

import (
	"dea"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var serviceKeys = []string{"name", "label", "tags", "plan", "plan_option", "credentials", "syslog_drain_url"}

type Env struct {
	envStrategy dea.EnvStrategy
	vcapApp     *map[string]interface{}
	vcapSrvcs   *map[string][]map[string]interface{}
}

func NewEnv(strategy dea.EnvStrategy) *Env {
	env := &Env{envStrategy: strategy}
	return env
}

func (e Env) ExportedSystemEnvironmentVariables() (string, error) {
	vcapApp, err := json.Marshal(e.VcapApplication())
	if err != nil {
		return "", err
	}
	vcapServices, err := json.Marshal(e.VcapServices())
	if err != nil {
		return "", err
	}

	envList := make([][]string, 0, 4)
	envList = append(envList, []string{"VCAP_APPLICATION", string(vcapApp)})
	envList = append(envList, []string{"VCAP_SERVICES", string(vcapServices)})

	memlimit := e.envStrategy.StartMessage().MemoryLimitMB()
	envList = append(envList, []string{"MEMORY_LIMIT", strconv.FormatUint(memlimit, 10) + "m"})

	services := e.envStrategy.StartMessage().Services()

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
	return to_export(translate_env(e.envStrategy.StartMessage().Environment()))
}

func (e Env) ExportedEnvironmentVariables() (string, error) {
	sysEnv, err := e.ExportedSystemEnvironmentVariables()
	if err != nil {
		return "", err
	}
	return sysEnv + e.ExportedUserEnvironmentVariables(), nil
}

func (e Env) VcapApplication() map[string]interface{} {
	if e.vcapApp == nil {
		msg := e.envStrategy.StartMessage()
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

func (e Env) VcapServices() map[string][]map[string]interface{} {
	if e.vcapSrvcs == nil {
		services := make(map[string][]map[string]interface{})
		servicesData := e.envStrategy.StartMessage().Services()

		for _, serviceInfo := range servicesData {
			serviceMap := make(map[string]interface{})
			for _, k := range serviceKeys {
				if v, exists := serviceInfo[k]; exists && (v != nil && v != "") {
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

func translate_env(env map[string]string) [][]string {
	envs := make([][]string, 0, len(env))
	for k, v := range env {
		envs = append(envs, []string{k, v})
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
