package main

import (
	"strconv"
)

const (
	COMMAND_QUEUE_SIZE = 10
)

type CommandQueuer interface {
	Next() *Command
	GetCommand() *Command
	GetCommandById(string) *Command
	Closed() bool
}

type CommandQueue struct {
	c        chan *Command
	next     *Command
	commands map[string]*Command
	nextId   int
	mutexId  chan bool
	closed   bool
}

func NewCommandQueue() *CommandQueue {
	return &CommandQueue{
		c:        make(chan *Command, COMMAND_QUEUE_SIZE),
		commands: make(map[string]*Command, COMMAND_QUEUE_SIZE),
		next:     nil,
		nextId:   0,
		mutexId:  make(chan bool, 1),
		closed:   false,
	}
}

// fetches the next command from queue, may return nil if none is available
func (cq *CommandQueue) Next() *Command {
	select {
	case cq.next = <-cq.c:
		if cq.next == nil {
			// channel was closed, stop listening for new commands
			cq.closed = true
		}
	default:
		cq.next = nil
	}
	return cq.next
}

// returns the current command, may return nil if none is available
func (cq *CommandQueue) GetCommand() *Command {
	if cq.next == nil {
		return cq.Next()
	}
	return cq.next
}

// fetch a command by id
func (cq *CommandQueue) GetCommandById(id string) *Command {
	return cq.commands[id]
}

// pushes a command into the queue
func (cq *CommandQueue) Enqueue(command *Command) {
	command.Id = cq.getNextId()
	cq.commands[command.Id] = command
	cq.c <- command
}

// closes the queue
func (cq *CommandQueue) Close() {
	close(cq.c)
}

// checks if the queue is closed AND empty
func (cq *CommandQueue) Closed() bool {
	return cq.closed
}

// create a unique task id
func (cq *CommandQueue) getNextId() string {
	cq.mutexId <- true
	cq.nextId++
	id := strconv.Itoa(cq.nextId)
	<-cq.mutexId
	return id
}
