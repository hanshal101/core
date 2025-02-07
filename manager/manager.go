package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"github.com/hanshal101/core/task"
	"github.com/hanshal101/core/worker"
)

// this is the manager model
// it will take all the requests from the api in a form of queue(FIFO)
// then two in-memory DB for storing the task and their events
// to make sure the manager knows all the workers in the cluster, hence storing it in an array
// mapping tasks to make the life of the manager easier for locating the task and managing their lifecycle
type Manager struct {
	Pending       queue.Queue
	TaskDB        map[uuid.UUID]*task.Task
	EventDB       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

func (m *Manager) GetTasks() []task.Task {
	tasks := make([]task.Task, 0, len(m.TaskDB))
	for _, t := range m.TaskDB {
		tasks = append(tasks, *t)
	}

	return tasks
}

func (m *Manager) SelectWorker() string {
	fmt.Println("This will select the best worker")
	var wrkr int
	if m.LastWorker == len(m.Workers)-1 {
		wrkr = 0
		m.LastWorker = 0
	} else {
		wrkr = m.LastWorker + 1
		m.LastWorker++
	}
	return m.Workers[wrkr]
}

func (m *Manager) updateTasks() {
	fmt.Println("This will update tasks")
	for _, wrkr := range m.Workers {
		url := fmt.Sprintf("http://%s/tasks", wrkr)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error in connecting to %v::%v\n", wrkr, err)
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Error in sending request (%d)::%v\n", resp.StatusCode, err)
		}
		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		if err := d.Decode(&tasks); err != nil {
			log.Printf("Error in decoding the response: Error: %v\n", err)
		}

		for _, t := range tasks {
			log.Printf("Updating Task :: %v\n", t)
			_, ok := m.TaskDB[t.ID]
			if !ok {
				log.Printf("Task not present: TaskID:%v :: Task:%v", t.ID, t)
				return
			}

			fmt.Println("st ----------------> ", m.TaskDB[t.ID].State, t.State)
			if m.TaskDB[t.ID].State != t.State {
				m.TaskDB[t.ID].State = t.State
			}

			m.TaskDB[t.ID].StartTime = t.StartTime
			m.TaskDB[t.ID].EndTime = t.EndTime
			m.TaskDB[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) SendWork() {
	fmt.Println("This will send work to the workers")
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()

		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		t := te.Task
		log.Printf("Pulled %v off pending queue\n", t)

		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], t.ID)
		m.TaskWorkerMap[t.ID] = w

		if te.Task.State != task.Completed {
			te.Task.State = task.Scheduled
		}

		m.TaskDB[t.ID] = &t
		m.EventDB[te.ID] = &te

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("Error in json marshal: Error: %v :: TaskEvent: %v\n", err, te)
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("Error in sending request to worker: Error: %v :: Url: %s\n", err, url)
			m.Pending.Enqueue(te)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			if err := d.Decode(&e); err != nil {
				fmt.Printf("Error in decoding resposne: Error: %v\n", err)
				return
			}
			log.Printf("Response error (%d): %s\n", e.HTTPStatusCode, e.Message)
			return
		}
		// t = task.Task{}
		// if err := d.Decode(&t); err != nil {
		// 	fmt.Printf("Error in decoding response: Error: %s", err)
		// 	return
		// }
		// log.Printf("%#v\n", t)
	} else {
		log.Println("No work in the queue")
	}
}

// Adding task
func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func New(workers []string) *Manager {
	workerTaskMap := make(map[string][]uuid.UUID)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		TaskDB:        make(map[uuid.UUID]*task.Task),
		EventDB:       make(map[uuid.UUID]*task.TaskEvent),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: make(map[uuid.UUID]string),
	}
}

// Updating task
func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking tasks for updates from workers!")
		m.updateTasks()
		log.Println("Tasks updates completed!")
		log.Println("Sleeping for 15 seconds!")
		time.Sleep(15 * time.Second)
	}
}

// Process tasks
func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing tasks!")
		m.SendWork()
		log.Println("Tasks processed!")
		log.Println("Sleeping for 10 seconds!")
		time.Sleep(10 * time.Second)
	}
}

// checking task health
func (m *Manager) checkTaskHealth(t task.Task) error {
	log.Printf("Calling health check for task: %v\n", t)

	w := m.TaskWorkerMap[t.ID]
	hostport := gethostport(t.HostPort)

	// Check if hostport is nil
	if hostport == nil {
		msg := fmt.Sprintf("No valid host port found for task: %v", t)
		log.Println(msg)
		return errors.New(msg)
	}

	worker := strings.Split(w, ":")

	// Construct the URL safely
	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostport, t.HealthCheck) // Dereference safely

	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error in health check: %v\n", err)
		log.Println(msg)
		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Health check failed: %v\n", resp.StatusCode)
		log.Println(msg)
		return errors.New(msg)
	}

	log.Printf("Health check passed for task: %v : With response: %v\n", t, resp)
	return nil
}

func gethostport(ports nat.PortMap) *string {
	for k, _ := range ports {
		// Ensure that we are returning a valid port
		if len(ports[k]) > 0 {
			return &ports[k][0].HostPort
		}
	}
	return nil // Return nil if no valid ports are found
}

// This will to all the health checks for the tasks
// Flow: 1. Call the manager’s checkTaskHealth method, which in turn will call the task’s health check endpoint
//  2. If the task’s health check fails, attempt to restart the task
//  3. If the task is in Failed state: attempt to restart the task
func (m *Manager) doHelathChecks() {
	for _, t := range m.TaskDB {
		if t.State == task.Running && t.RestartCount < 3 {
			if err := m.checkTaskHealth(*t); err != nil {
				if t.RestartCount < 3 {
					m.restartTask(t)
				}
			} else if t.State == task.Failed && t.RestartCount < 3 {
				m.restartTask(t)
			}
		}
	}
}

// this will restart the task
func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDB[t.ID] = t

	te := task.TaskEvent{
		ID:    uuid.New(),
		State: task.Running,
		Task:  *t,
	}

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Unable to marshal the task event: %v\n", err)
	}

	url := fmt.Sprintf("http://%v/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v : Error: %v", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrResponse{}
		if err := d.Decode(&e); err != nil {
			log.Printf("Error in decoding response: %v\n", err)
			return
		}
		log.Printf("Response error : (%d): %v", e.HTTPStatusCode, e.Message)
		return
	}
	log.Printf("%#v\n", t)
}

func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Performing task health check")
		m.doHelathChecks()
		log.Println("Task health checks completed")
		log.Println("Sleeping for 60 seconds")
		time.Sleep(10 * time.Second)
	}
}

// Creating manager API

type API struct {
	Address string
	Port    int
	Manager *Manager
	Router  *gin.Engine
}

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}

func (a *API) StartTask(c *gin.Context) {
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	if err := d.Decode(&te); err != nil {
		var fieldError *json.UnmarshalTypeError
		if errors.As(err, &fieldError) {
			msg := fmt.Sprintf("invalid type for field %s: expected %s but got %s",
				fieldError.Field, fieldError.Type, fieldError.Value)
			log.Println(msg)
			c.JSON(http.StatusBadRequest, ErrResponse{HTTPStatusCode: http.StatusBadRequest, Message: msg})
			return
		}

		msg := fmt.Sprintf("error in unmarshalling body: %v", err)
		log.Println(msg)
		c.JSON(http.StatusBadRequest, ErrResponse{HTTPStatusCode: http.StatusBadRequest, Message: msg})
		return
	}

	a.Manager.AddTask(te)
	c.Status(http.StatusCreated)
}

func (a *API) GetTasks(c *gin.Context) {
	tasks := a.Manager.GetTasks()
	if len(tasks) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (a *API) GetTasksbyID(c *gin.Context) {
	tasks := a.Manager.GetTasks()
	if len(tasks) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	tID := c.Param("taskID")
	taskID, err := uuid.Parse(tID)
	if err != nil {
		fmt.Println("Error parsing UUID:", err)
		return
	}
	var t task.Task
	for _, tk := range tasks {
		if tk.ID == taskID {
			t = tk
		}
	}
	c.JSON(http.StatusOK, t)
}

func (a *API) StopTask(c *gin.Context) {
	tID := c.Param("taskID")
	utID, _ := uuid.Parse(tID)

	_, ok := a.Manager.TaskDB[utID]
	if !ok {
		log.Printf("task does not exists, uuid: %v", utID)
		c.Status(http.StatusNotFound)
	}
	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
	}
	taskToStop := a.Manager.TaskDB[utID]
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy
	a.Manager.AddTask(te)

	log.Printf("task with uuid %v has been stopped: %v", utID, taskCopy)
	c.Status(http.StatusNoContent)
}

func (a *API) InitRouter() {
	// tasks
	a.Router.GET("/tasks", a.GetTasks)
	a.Router.GET("/tasks/:taskID", a.GetTasksbyID)
	a.Router.POST("/tasks", a.StartTask)
	a.Router.DELETE("/tasks/:taskID", a.StopTask)
}

func (a *API) Start() {
	a.InitRouter()
	a.Router.Run(fmt.Sprintf("%s:%v", a.Address, a.Port))
}
