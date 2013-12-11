package task

import (
	"dea/container"
	"dea/utils"
	steno "github.com/cloudfoundry/gosteno"
	"os"
	"time"
)

type TaskPromises interface {
	Promise_stop() error
	Promise_destroy() error
}

type taskPromises struct {
	task *Task
}

type Task struct {
	Container container.Container
	Logger    *steno.Logger
	TaskPromises
	bindMounts []map[string]string
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

func (t *Task) Destroy(callback func(error) error) {
	destroy_promise := func() error {
		t.Logger.Info("task.destroying")
		return t.Promise_destroy()
	}

	t.ResolveAndLog(destroy_promise, "task.destroy", func(e error) error {
		if callback != nil {
			return callback(e)
		}
		return nil
	})
}

func (t *Task) Copy_out_request(source_path, destination_path string) error {
	os.MkdirAll(destination_path, 0755)
	err := t.Container.CopyOut(source_path, destination_path, os.Getuid())
	if err != nil {
		t.Logger.Warnf("Error copying files out of Container: %s", err.Error())
	}
	return err
}

func (t *Task) ResolveAndLog(p utils.Promise, name string, callback func(error) error) {
	start := time.Now()
	utils.Async_promise(p, func(err error) error {
		if callback != nil {
			err = callback(err)
		}

		duration := time.Now().Sub(start)
		if err != nil {
			t.Logger.Warnd(map[string]interface{}{
				"error":    err,
				"duration": duration,
			}, name+".failed")
		} else {
			t.Logger.Infod(map[string]interface{}{
				"duration": duration,
			}, name+".completed")
		}

		return nil
	})
}

func (p *taskPromises) Promise_stop() error {
	return p.task.Container.Stop()
}

func (p *taskPromises) Promise_destroy() error {
	p.task.Container.Destroy()
	return nil
}
