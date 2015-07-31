package main

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type Command struct {
	Id           string
	Cmd          string
	Status       *mesos.TaskStatus
	StdoutPailer *Pailer
	StderrPailer *Pailer
}

func (c *Command) StopPailers() {
	if c.StdoutPailer != nil {
		c.StdoutPailer.Stop()
		c.StdoutPailer = nil
	}
	if c.StderrPailer != nil {
		c.StderrPailer.Stop()
	}
}

func (c *Command) WaitForPailers() {
	if c.StdoutPailer != nil {
		c.StdoutPailer.Wait()
		c.StdoutPailer = nil
	}
	if c.StderrPailer != nil {
		c.StderrPailer.Wait()
		c.StderrPailer = nil
	}
}
