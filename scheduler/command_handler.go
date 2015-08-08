package main

import (
	"os"

	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type CommandHandler struct {
	commands      []*Command
	frameworkId   *mesos.FrameworkID
	tasksLaunched int
	tasksEnded    int
	tasksFailed   int
	totalTasks    int
}

func NewCommandHandler() *CommandHandler {
	return &CommandHandler{
		commands:      []*Command{},
		frameworkId:   nil,
		tasksLaunched: 0,
		tasksEnded:    0,
		tasksFailed:   0,
		totalTasks:    0,
	}
}

func (ch *CommandHandler) SetFrameworkId(id *mesos.FrameworkID) {
	ch.frameworkId = id
}

func (ch *CommandHandler) CommandLaunched(c *Command) {
	ch.tasksLaunched++
	ch.commands = append(ch.commands, c)
}

func (ch *CommandHandler) CommandRunning(c *Command) {
	c.StdoutPailer = ch.createAndStartPailer(c, "cmd.stdout", os.Stdout)
	c.StderrPailer = ch.createAndStartPailer(c, "cmd.stderr", os.Stderr)
}

func (ch *CommandHandler) CommandEnded(c *Command) {
	ch.tasksEnded++
	c.StopPailers()
}

func (ch *CommandHandler) CommandFinished(c *Command) {

}

func (ch *CommandHandler) CommandFailed(c *Command) {
	ch.tasksFailed++
}

func (ch *CommandHandler) FinishAllCommands() {
	for _, c := range ch.commands {
		c.WaitForPailers()
	}
}

func (ch *CommandHandler) HasFailures() bool {
	return ch.tasksFailed > 0
}

func (ch *CommandHandler) HasRunningTasks() bool {
	return ch.tasksLaunched > ch.tasksEnded
}

// private

func (ch *CommandHandler) createAndStartPailer(c *Command, file string, w StringWriter) *Pailer {
	p, err := NewPailer(w, master, ch.frameworkId, c, file)
	if err != nil {
		log.Errorf("Unable to start pailer for task %s: %s\n", c.Id, err)
		return nil
	} else {
		p.Start()
		return p
	}
}
