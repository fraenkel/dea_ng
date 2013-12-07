package starting

import dstarting "dea/starting"

type FakePromises struct {
	DestroyCallback func()
}

func (f *FakePromises) Promise_start() error {
	return nil
}
func (f *FakePromises) Promise_copy_out() error {
	return nil
}
func (f *FakePromises) Promise_crash_handler() error {
	return nil
}
func (f *FakePromises) Promise_container() error {
	return nil
}
func (f *FakePromises) Promise_droplet() error {
	return nil
}
func (f *FakePromises) Promise_exec_hook_script(key string) error {
	return nil
}
func (f *FakePromises) Promise_state(from []dstarting.State, to dstarting.State) error {
	return nil
}
func (f *FakePromises) Promise_extract_droplet() error {
	return nil
}
func (f *FakePromises) Promise_setup_environment() error {
	return nil
}
func (f *FakePromises) Link() {
}
func (f *FakePromises) Promise_read_instance_manifest(container_path string) (map[string]interface{}, error) {
	return nil, nil
}
func (f *FakePromises) Promise_health_check() (bool, error) {
	return true, nil
}

func (f *FakePromises) Promise_stop() error {
	return nil
}
func (f *FakePromises) Promise_destroy() {
	if f.DestroyCallback != nil {
		f.DestroyCallback()
	}
}
