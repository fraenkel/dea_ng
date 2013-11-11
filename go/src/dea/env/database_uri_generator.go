package env

import (
	"net/url"
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

func NewDatabaseUriGenerator(services []map[string]interface{}) *DatabaseUriGenerator {
	return &DatabaseUriGenerator{services: services}
}

func (dbGen *DatabaseUriGenerator) DatabaseUri() (*url.URL, error) {
	uri, err := dbGen.boundDatabaseUri()
	if err != nil {
		return nil, err
	}

	if uri != nil {
		if newScheme, exists := rails_scheme[uri.Scheme]; exists {
			uri.Scheme = newScheme
		}
	}

	return uri, nil
}

func (dbGen *DatabaseUriGenerator) boundDatabaseUri() (*url.URL, error) {
	boundDBs, err := dbGen.boundRelationalValidDatabases()
	if err != nil {
		return nil, err
	}

	if len(boundDBs) > 0 {
		return boundDBs[0].uri, nil
	}

	return nil, nil
}

func (dbGen *DatabaseUriGenerator) boundRelationalValidDatabases() ([]serviceBinding, error) {
	if dbGen.boundDatabases == nil {
		collection := make([]serviceBinding, 0, 1)
		for _, binding := range dbGen.services {
			if b := binding["credentials"]; b != nil {
				var creds_uri string
				if creds, ok := b.(map[string]interface{}); ok {
					creds_uri = creds["uri"].(string)
				} else if creds, ok := b.(map[string]string); ok {
					creds_uri = creds["uri"]
				}

				uri, err := url.Parse(creds_uri)
				if err != nil {
					return nil, err
				}

				if valid_db_type(uri.Scheme) {
					name, _ := binding["name"].(string)
					collection = append(collection, serviceBinding{
						uri: uri, name: name,
					})
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
