package router_client

import (
	"dea"
	"github.com/cloudfoundry/yagnats"
)

type FakeRouterClient struct {
	RegisterInstanceInstance   dea.Instance
	UnregisterInstanceInstance dea.Instance
}

func (frc *FakeRouterClient) RegisterInstance(i dea.Instance, opts map[string]interface{}) error {
	frc.RegisterInstanceInstance = i
	return nil
}
func (frc *FakeRouterClient) UnregisterInstance(i dea.Instance, opts map[string]interface{}) error {
	frc.UnregisterInstanceInstance = i
	return nil
}
func (frc *FakeRouterClient) Greet(callback yagnats.Callback) error {
	return nil
}
func (frc *FakeRouterClient) Register_directory_server(host string, port uint32, uri string) error {
	return nil
}
func (frc *FakeRouterClient) Unregister_directory_server(host string, port uint32, uri string) error {
	return nil
}
