package boot

import (
	"dea"
)

type FakeInstanceManager struct {
	CreateCallback     func(attributes map[string]interface{}) dea.Instance
	StartAppInvoked    bool
	StartAppAttributes map[string]interface{}
}

func (fim *FakeInstanceManager) CreateInstance(attributes map[string]interface{}) dea.Instance {
	if fim.CreateCallback != nil {
		return fim.CreateCallback(attributes)
	}
	return nil
}
func (fim *FakeInstanceManager) StartApp(attributes map[string]interface{}) {
	fim.StartAppAttributes = attributes
	fim.StartAppInvoked = true
}
