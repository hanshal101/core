package worker

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/task"
)

// this is a worker model
// so it has following duties to do: run containers, accept task from manager, provide stats and keep track of tasks state
// for running containers and keeping track of the state we can store it on map which can then implemented to etcd
// since we would implement the task in a queue(FIFO) we would do this with normal golang-collections library
// at last keeping the count of the task in the queue as task-count
type Worker struct {
	Name      string
	Queue     queue.Queue
	DB        map[uuid.UUID]task.Task
	TaskCount int
}

func (w *Worker) GetStats() {
	fmt.Println("This will collect stats from worker")
}

// this is diff from StartTask
// as this is responsible for identifying the task’s current state and then either starting or stopping
func (w *Worker) RunTask() {
	fmt.Println("This will run tasks")
}

func (w *Worker) StartTask() {
	fmt.Println("This will start tasks")
}

func (w *Worker) StopTask() {
	fmt.Println("This will stop tasks")
}
