package router_client

import (
	"dea/starting"
	"github.com/cloudfoundry/yagnats"
)

type FakeRouterClient struct {
	RegisterInstanceInstance   *starting.Instance
	UnregisterInstanceInstance *starting.Instance
}

func (frc *FakeRouterClient) RegisterInstance(i *starting.Instance, opts map[string]interface{}) error {
	frc.RegisterInstanceInstance = i
	return nil
}
func (frc *FakeRouterClient) UnregisterInstance(i *starting.Instance, opts map[string]interface{}) error {
	frc.UnregisterInstanceInstance = i
	return nil
}
func (frc *FakeRouterClient) Greet(callback yagnats.Callback) error {
	return nil
}
func (frc *FakeRouterClient) Register_directory_server(host string, port uint16, uri string) error {
	return nil
}
func (frc *FakeRouterClient) Unregister_directory_server(host string, port uint16, uri string) error {
	return nil
}
