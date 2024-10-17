package task

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
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

// model to run a container will sufficient configuration
type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	Cmd           []string
	Image         string
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

// the docker model with the docker client and th	 configuration of the container to run
type Docker struct {
	Client      *client.Client
	Config      Config
	ContainerID string
}

// this will be used as a result after the task is assigned to analyze whether the docker container of executed sucessfully or not
type DockerResult struct {
	Error       error
	Action      string
	ContainerID string
	Result      string
}

// This is similiar to docker run, stop, rm command
func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(
		ctx, d.Config.Image, image.PullOptions{},
	)
	if err != nil {
		log.Printf("Error in pulling the image %s: %v\n", d.Config, err)
		return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	r := container.Resources{
		Memory: d.Config.Memory,
	}

	cc := container.Config{
		Image: d.Config.Image,
		Env:   d.Config.Env,
	}

	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(
		ctx, &cc, &hc, nil, nil, d.Config.Name,
	)
	if err != nil {
		log.Printf("Error in creating container %s: %v", d.Config, err)
		return DockerResult{Error: err}
	}

	if err := d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf("Error in starting the container %s: %v", d.Config, err)
		return DockerResult{Error: err}
	}

	d.ContainerID = resp.ID

	out, err := d.Client.ContainerLogs(
		ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		log.Printf("Error in getting container logs %s: %v", d.Config, err)
		return DockerResult{Error: err}
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		Error:       nil,
		Action:      "start",
		ContainerID: resp.ID,
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container: %s", id)
	ctx := context.Background()
	if err := d.Client.ContainerStop(
		ctx, id, container.StopOptions{},
	); err != nil {
		log.Printf("Error in stopping container %s: %v", d.Config, err)
		return DockerResult{Error: err}
	}

	if err := d.Client.ContainerRemove(
		ctx, id, container.RemoveOptions{},
	); err != nil {
		log.Printf("Error in removing container %s: %v", d.Config, err)
		return DockerResult{Error: err}
	}

	return DockerResult{
		Error:       nil,
		Action:      "stop",
		ContainerID: id,
		Result:      "success",
	}
}
