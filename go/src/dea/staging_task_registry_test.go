package dea

import (
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestRegister(t *testing.T) {
    stagingRegistry := NewStagingTaskRegistry()
    stagingTask := NewStagingTask(map[string]string{"task_id": "task1"})
    stagingRegistry.Register(stagingTask)
    assert.Equal(t, len(stagingRegistry.tasks), 1)
}

func TestUnregisterWhenPreviouslyRegistered(t *testing.T) {
    stagingRegistry := NewStagingTaskRegistry()
    stagingTask := NewStagingTask(map[string]string{"task_id": "task1"})
    stagingRegistry.Register(stagingTask)

    stagingRegistry.Unregister(stagingTask)
    assert.Equal(t, len(stagingRegistry.tasks), 0)
}

func TestUnregisterWhenNotRegistered(t *testing.T) {
    stagingRegistry := NewStagingTaskRegistry()
    stagingTask := NewStagingTask(map[string]string{"task_id": "task1"})

    stagingRegistry.Unregister(stagingTask)
    assert.Equal(t, len(stagingRegistry.tasks), 0)
}

func TestTaskRetrieval(t *testing.T) {
    stagingRegistry := NewStagingTaskRegistry()
    stagingTask := NewStagingTask(map[string]string{"task_id": "task1"})
    stagingRegistry.Register(stagingTask)

    assert.Equal(t, stagingRegistry.Task(stagingTask.id), stagingTask)
}

func TestEach(t *testing.T) {
    stagingRegistry := NewStagingTaskRegistry()
    stagingTask1 := NewStagingTask(map[string]string{"task_id": "task1"})
    stagingTask2 := NewStagingTask(map[string]string{"task_id": "task2"})
    stagingRegistry.Register(stagingTask1)
    stagingRegistry.Register(stagingTask2)

    var i int = 0
    stagingRegistry.Each(func(*StagingTask) {
        i++
    })
    assert.Equal(t, 2, i)
}
