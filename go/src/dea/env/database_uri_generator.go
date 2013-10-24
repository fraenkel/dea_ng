package env

import (
	"errors"
	"net/url"
	"strings"
)

var (
	valid_db_types = []string{"mysql", "mysql2", "postgres", "postgresql"}
	rails_scheme   = map[string]string{
		"mysql":      "mysql2",
		"postgresql": "postgres",
	}
)

type DatabaseUriGenerator struct {
	services       []map[string]interface{}
	boundDatabases []serviceBinding
}

type serviceBinding struct {
	name string
	uri  *url.URL
}

func NewDatabaseUriGenerator(services []map[string]interface{}) DatabaseUriGenerator {
	return DatabaseUriGenerator{services: services}
}

func (dbGen DatabaseUriGenerator) DatabaseUri() (*url.URL, error) {
	uri, err := dbGen.boundDatabaseUri()
	if err != nil {
		return nil, err
	}

	if newScheme, exists := rails_scheme[uri.Scheme]; exists {
		uri.Scheme = newScheme
	}
	return uri, nil
}

func (dbGen DatabaseUriGenerator) boundDatabaseUri() (*url.URL, error) {
	boundDBs, err := dbGen.boundRelationalValidDatabases()
	if err != nil {
		return nil, err
	}

	switch len(boundDBs) {
	case 0:
		return nil, nil
	case 1:
		return boundDBs[0].uri, nil
	default:
		for _, binding := range boundDBs {
			if strings.HasSuffix(binding.name, "production") || strings.HasSuffix(binding.name, "prod") {
				return binding.uri, nil
			}
			return nil, errors.New("Unable to determine primary database from multiple. Please bind only one database service to Rails applications.")
		}
	}

	return nil, nil
}

func (dbGen DatabaseUriGenerator) boundRelationalValidDatabases() ([]serviceBinding, error) {
	if dbGen.boundDatabases == nil {
		collection := make([]serviceBinding, 1)
		for _, binding := range dbGen.services {
			if b := binding["credentials"]; b != nil {
				if creds, ok := b.(map[string]string); ok {
					uri, err := url.Parse(creds["uri"])
					if err != nil {
						return nil, err
					}
					if valid_db_type(uri.Scheme) {
						collection = append(collection, serviceBinding{
							uri: uri, name: binding["name"].(string),
						})
					}
				}
			}
		}
		dbGen.boundDatabases = collection
	}

	return dbGen.boundDatabases, nil
}

func valid_db_type(scheme string) bool {
	for _, v := range valid_db_types {
		if v == scheme {
			return true
		}
	}

	return false
}
