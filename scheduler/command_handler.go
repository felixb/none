package main

type CommandHandler struct {
	commands      []*Command
	tasksLaunched int
	tasksEnded    int
	tasksFailed   int
	totalTasks    int
}

func NewCommandHandler() *CommandHandler {
	return &CommandHandler{
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
