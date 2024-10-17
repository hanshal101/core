package main

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/manager"
	"github.com/hanshal101/core/node"
	"github.com/hanshal101/core/task"
	"github.com/hanshal101/core/worker"
)

func main() {
	t := task.Task{
		ID:     uuid.New(),
		Name:   "Test-1",
		State:  task.Pending,
		Image:  "Image-Test-1",
		Memory: 1024,
		Disk:   1,
	}
	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Pending,
		Timestamp: time.Now(),
		Task:      t,
	}
	fmt.Printf("task: %v\n", t)
	fmt.Printf("task event: %v\n", te)

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]task.Task),
	}

	fmt.Printf("worker: %v\n", w)

	w.GetStats()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: *queue.New(),
		TaskDB:  make(map[string][]task.Task),
		EventDB: make(map[string][]task.TaskEvent),
		Workers: []string{w.Name},
	}

	fmt.Printf("manager: %v\n", m)
	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name:   "Node-1",
		IP:     "192.168.1.1",
		Cores:  4,
		Memory: 1024,
		Disk:   25,
		Role:   "worker",
	}
	fmt.Printf("node: %v\n", n)

	fmt.Println("creating a container")
	dockerTask, createTask := createContainer()
	if createTask.Error != nil {
		fmt.Printf(createTask.Error.Error())
		os.Exit(1)
	}

	fmt.Println("Sleeping for 10 Seconds")
	time.Sleep(10 * time.Second)

	rmTask := stopContainer(dockerTask, createTask.ContainerID)
	if rmTask.Error != nil {
		fmt.Printf(rmTask.Error.Error())
		os.Exit(1)
	}
}

// testing contatiner start & stop
func createContainer() (*task.Docker, *task.DockerResult) {
	c := task.Config{
		Name:  "container-1",
		Image: "postgres:alpine",
		Env: []string{
			"POSTGRES_USER=core",
			"POSTGRES_PASSWORD=core",
		},
	}

	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := task.Docker{
		Client: dc,
		Config: c,
	}

	result := d.Run()
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil, nil
	}
	fmt.Printf("Container %s is running with config %v\n", result.ContainerID, d.Config)
	return &d, &result
}

func stopContainer(d *task.Docker, id string) *task.DockerResult {
	result := d.Stop(id)
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil
	}
	fmt.Printf("Container %s is stopped with config %v\n", result.ContainerID, d.Config)
	return &result

}
