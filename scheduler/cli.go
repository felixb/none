package main

import (
	"bufio"
	"os"

	mesos "github.com/mesos/mesos-go/mesosproto"
)

type Cli struct {
	queue         *CommandQueue
	containerInfo *mesos.ContainerInfo
	uris          []*mesos.CommandInfo_URI
	cpuReq        *float64
	memReq        *float64
}

func NewCli(queue *CommandQueue, containerInfo *mesos.ContainerInfo, uris []*mesos.CommandInfo_URI, cpuReq, memReq *float64) *Cli {
	return &Cli{
		queue:         queue,
		containerInfo: containerInfo,
		uris:          uris,
		cpuReq:        cpuReq,
		memReq:        memReq,
	}
}

// write promt, start reading commands from stdin...
func (cli *Cli) Start() {
	// TODO print promt and stuff
	reader := bufio.NewReader(os.Stdin)
	cmd, err := reader.ReadString('\n')
	for err == nil {
		cli.StartCommand(&cmd)
		cmd, err = reader.ReadString('\n')
	}
	cli.Close()
}

func (cli *Cli) StartCommand(cmd *string) {
	cli.queue.Enqueue(&Command{
		Cmd:           *cmd,
		CpuReq:        *cli.cpuReq,
		MemReq:        *cli.memReq,
		ContainerInfo: cli.containerInfo,
		Uris:          cli.uris,
	})
}

func (cli *Cli) Close() {
	cli.queue.Close()
}
