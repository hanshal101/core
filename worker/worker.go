package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	DB        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) GetStats() {
	fmt.Println("This will collect stats from worker")
}

// this is diff from StartTask
// as this is responsible for identifying the task’s current state and then either starting or stopping
func (w *Worker) RunTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks in the queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskPersisted := w.DB[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.DB[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidStateTransitions(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we can't apply this")
		}
	} else {
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted, taskQueued)
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Run()
	if result.Error != nil {
		log.Printf("Error in Running the container %v: %v\n", d.Config, result.Error)
		t.State = task.Failed
		w.DB[t.ID] = &t
		return result
	}
	t.ContainerID = result.ContainerID
	t.State = task.Running
	w.DB[t.ID] = &t

	log.Printf("Running the container %v: %v\n", d.Config, &t)
	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("Error in Stopping the container %v: %v\n", d.Config, result.Error)
		t.State = task.Failed
		w.DB[t.ID] = &t
		return result
	}
	t.EndTime = time.Now().UTC()
	t.State = task.Completed
	w.DB[t.ID] = &t

	log.Printf("Stopped and removed the container %v: %v\n", d.Config, &t)
	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []task.Task {
	tasks := make([]task.Task, 0, len(w.DB))
	for _, t := range w.DB {
		tasks = append(tasks, *t)
	}

	return tasks
}

type API struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *gin.Engine
}

func (a *API) StartTask(c *gin.Context) {
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	if err := d.Decode(&te); err != nil {
		msg := fmt.Sprintf("error in unmarshalling body: %v", err)
		log.Println(msg)
		c.JSON(http.StatusBadRequest, gin.H{"message": msg})
		return
	}

	a.Worker.AddTask(te.Task)
	c.Status(http.StatusCreated)
}

func (a *API) GetTasks(c *gin.Context) {
	tasks := a.Worker.GetTasks()
	if len(tasks) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (a *API) DeleteTask(c *gin.Context) {
	tID := c.Param("taskID")
	utID, _ := uuid.Parse(tID)

	_, ok := a.Worker.DB[utID]
	if !ok {
		log.Printf("task does not exists, uuid: %v", utID)
		c.Status(http.StatusNotFound)
	}
	taskToStop := a.Worker.DB[utID]
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	log.Printf("task with uuid %v has been stopped: %v", utID, taskCopy)
	c.Status(http.StatusNoContent)
}

func (a *API) InitRouter() {
	a.Router.GET("/tasks", a.GetTasks)
	a.Router.POST("/tasks", a.StartTask)
	a.Router.DELETE("/tasks/:taskID", a.DeleteTask)
}

func (a *API) Start() {
	a.InitRouter()
	a.Router.Run(fmt.Sprintf("%s:%v", a.Address, a.Port))
}
