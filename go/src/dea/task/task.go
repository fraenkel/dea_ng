package task

import (
	"dea/container"
	"dea/utils"
	"os"
)

type Task struct {
	container *container.Container
}

func NewTask() Task {
	return Task{}
}

func (t Task) Container() *container.Container {
	if t.container == nil {
		container := container.NewContainer(t.wardenSocket())
		t.container = &container
	}
	return t.container
}

func (t *Task) wardenSocket() string {
	panic("Not Implemented")
}

func (t *Task) Start(func()) {
	panic("Not Implemented")
}

func (t *Task) Promise_stop() error {
	return t.Container().Stop()
}

func (t *Task) Destroy() {
	utils.Logger("Task").Info("task.destroying")
	t.Promise_destroy()
	utils.Logger("Task").Info("task.destroy.completed")
}

func (t *Task) Copy_out_request(source_path, destination_path string) error {
	os.MkdirAll(destination_path, 0755)
	err := t.Container().CopyOut(source_path, destination_path, os.Getuid())
	if err != nil {
		utils.Logger("Task").Warnf("Error copying files out of container: %s", err.Error())
	}
	return err
}

func (t *Task) Promise_destroy() {
	t.Container().Destroy()
}
