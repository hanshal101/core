package task

import (
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type State int

// iota represents the diff stages of the task
const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

// since this is a basic implementation of a container orchestrator
// task here only contains the image name, the uuid, and the state
// to find the best worker in our cluster for the application we check this by viewing memory and disk
// also here the restart-policy is same as implemented in kubernetes while exposed-ports and port-bindings are like services
// start-time and end-time looks cool to show in the CLI
type Task struct {
	ID            uuid.UUID
	Name          string
	State         State
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	EndTime       time.Time
}

// if user wants to stop a task it can do through task-event
type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}
