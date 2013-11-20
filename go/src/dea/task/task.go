package task

import (
	"dea/container"
	steno "github.com/cloudfoundry/gosteno"
	"os"
)

type TaskPromises interface {
	Promise_stop() error
	Promise_destroy()
}

type taskPromises struct {
	task *Task
}

type Task struct {
	Container container.Container
	Logger    *steno.Logger
	TaskPromises
}

func NewTask(wardenSocket string, logger *steno.Logger) *Task {
	p := &taskPromises{}
	t := &Task{
		Container:    container.NewContainer(wardenSocket),
		Logger:       logger,
		TaskPromises: p,
	}

	p.task = t
	return t
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

func (p *taskPromises) Promise_stop() error {
	return p.task.Container.Stop()
}

func (p *taskPromises) Promise_destroy() {
	p.task.Container.Destroy()
}
