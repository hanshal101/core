package manager

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/task"
)

// this is the manager model
// it will take all the requests from the api in a form of queue(FIFO)
// then two in-memory DB for storing the task and their events
// to make sure the manager knows all the workers in the cluster, hence storing it in an array
// mapping tasks to make the life of the manager easier for locating the task and managing their lifecycle
type Manager struct {
	Pending       queue.Queue
	TaskDB        map[string][]task.Task
	EventDB       map[string][]task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

func (m *Manager) SelectWorker() {
	fmt.Println("This will select the best worker")
}

func (c *Manager) UpdateTasks() {
	fmt.Println("This will update tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("This will send work to the workers")
}
