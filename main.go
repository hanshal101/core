package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/manager"
	"github.com/hanshal101/core/task"
	"github.com/hanshal101/core/worker"
)

func main() {
	whost := "0.0.0.0"
	wport := 50051

	mhost := "0.0.0.0"
	mport := 50050

	fmt.Println("starting core worker")

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}

	wapi := worker.API{
		Address: whost,
		Port:    wport,
		Worker:  &w,
		Router:  gin.Default(),
	}

	go w.RunTasks()
	go w.CollectStats()
	go wapi.Start()

	fmt.Println("Sleeping for 10 seconds to start the worker api")
	time.Sleep(10 * time.Second)

	fmt.Println("Starting core manager")
	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)

	mapi := manager.API{
		Address: mhost,
		Port:    mport,
		Manager: m,
		Router:  gin.Default(),
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()
	mapi.Start()

	// println("Sleeping")
	// time.Sleep(15 * time.Second)

	// for i := 0; i < 3; i++ {
	// 	t := task.Task{
	// 		ID:    uuid.New(),
	// 		Name:  fmt.Sprintf("test-container-%d", i),
	// 		Image: "ubuntu:latest",
	// 	}
	// 	te := task.TaskEvent{
	// 		ID:   uuid.New(),
	// 		Task: t,
	// 	}
	// 	m.AddTask(te)
	// 	m.SendWork()
	// }
	// go func() {
	// 	for {
	// 		fmt.Printf("[Manager] :: Updating tasks from %d worker\n", m.LastWorker)
	// 		m.UpdateTasks()
	// 		time.Sleep(5 * time.Second)
	// 	}
	// }()

	// for {
	// 	for _, t := range m.TaskDB {
	// 		fmt.Printf("[Manager] :: TaskID: %d, State: %d, Task: %v\n", t.ID, t.State, t)
	// 		time.Sleep(15 * time.Second)
	// 	}
	// }
}
