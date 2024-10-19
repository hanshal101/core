//

package main

import (
	"fmt"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/task"
	"github.com/hanshal101/core/worker"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
		Queue: *queue.New(),
		DB:    db,
	}
	t := task.Task{
		ID:    uuid.New(),
		Name:  "task-test",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}
	fmt.Println("starting task")
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
	t.ContainerID = result.ContainerID
	fmt.Printf("task %v is running in container %s\n", t, result)
	fmt.Println("SLEEP for 30 Seconds")
	time.Sleep(30 * time.Second)

	fmt.Println("stopping task")
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
}
