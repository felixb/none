package main

import (
	log "github.com/golang/glog"
)

type CommandHandler struct {
	downloadFiles *string
	commands      []*Command
	tasksLaunched int
	tasksEnded    int
	tasksFailed   int
	totalTasks    int
}

func NewCommandHandler(downloadFiles *string) *CommandHandler {
	return &CommandHandler{
		downloadFiles: downloadFiles,
		commands:      []*Command{},
		tasksLaunched: 0,
		tasksEnded:    0,
		tasksFailed:   0,
		totalTasks:    0,
	}
}

func (ch *CommandHandler) CommandLaunched(c *Command) {
	ch.tasksLaunched++
	ch.commands = append(ch.commands, c)
}

func (ch *CommandHandler) CommandRunning(c *Command) {
	c.StartPailers()
}

func (ch *CommandHandler) CommandEnded(c *Command) {
	ch.tasksEnded++
	c.StopPailers()
	if ch.downloadFiles != nil && *ch.downloadFiles != "" {
		// TODO do thins async?
		err := c.DownloadFile(ch.downloadFiles)
		if err != nil {
			log.Errorln("Failed loading file '", *ch.downloadFiles, "'", err)
			// TODO do something?
		}
	}
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
