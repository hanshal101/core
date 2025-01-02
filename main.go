package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/manager"
	"github.com/hanshal101/core/task"
	"github.com/hanshal101/core/worker"
)

func main() {
	host := "0.0.0.0"
	port := 50051
	fmt.Println("starting core worker")

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.API{
		Address: host,
		Port:    port,
		Worker:  &w,
		Router:  gin.Default(),
	}

	go runTasks(&w)
	go w.CollectStats()
	go api.Start()
	println("Sleeping")
	time.Sleep(15 * time.Second)

	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := manager.New(workers)
	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			Image: "ubuntu:latest",
		}
		te := task.TaskEvent{
			ID:   uuid.New(),
			Task: t,
		}
		m.AddTask(te)
		m.SendWork()
	}
	go func() {
		for {
			fmt.Printf("[Manager] :: Updating tasks from %d worker\n", m.LastWorker)
			m.UpdateTasks()
			time.Sleep(5 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDB {
			fmt.Printf("[Manager] :: TaskID: %d, State: %d, Task: %v\n", t.ID, t.State, t)
			time.Sleep(15 * time.Second)
		}
	}
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("error running tasks: %v", result.Error)
			}
		} else {
			log.Println("no tasks in the queue")
		}
		log.Println("Sleeping for 5 seconds")
		time.Sleep(5 * time.Second)
	}
}
