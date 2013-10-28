package task

import (
	"dea/container"
	steno "github.com/cloudfoundry/gosteno"
	"os"
)

type Task struct {
	Container *container.Container
	Logger    *steno.Logger
}

func NewTask(wardenSocket string, logger *steno.Logger) Task {
	return Task{
		Container: container.NewContainer(wardenSocket),
		Logger:    logger}
}

func (t *Task) Promise_stop() error {
	return t.Container.Stop()
}

func (t *Task) Destroy() {
	t.Logger.Info("task.destroying")
	t.Promise_destroy()
	t.Logger.Info("task.destroy.completed")
}

func (t *Task) Copy_out_request(source_path, destination_path string) error {
	os.MkdirAll(destination_path, 0755)
	err := t.Container.CopyOut(source_path, destination_path, os.Getuid())
	if err != nil {
		t.Logger.Warnf("Error copying files out of Container: %s", err.Error())
	}
	return err
}

func (t *Task) Promise_destroy() {
	t.Container.Destroy()
}
