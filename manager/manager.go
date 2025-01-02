package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

func (m *Manager) UpdateTasks() {
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

		te.Task.State = task.Scheduled
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
