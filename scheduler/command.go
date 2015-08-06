package main

import (
	"fmt"

	mesos "github.com/mesos/mesos-go/mesosproto"
)

type Command struct {
	Id            string
	Cmd           string
	CpuReq        float64
	MemReq        float64
	ContainerInfo *mesos.ContainerInfo
	Uris          []*mesos.CommandInfo_URI
	Status        *mesos.TaskStatus
	StdoutPailer  *Pailer
	StderrPailer  *Pailer
}

func (c *Command) GetCommandInfo() *mesos.CommandInfo {
	value := "sh"
	shell := false
	var args []string

	if c.ContainerInfo == nil {
		args = []string{"", "-c", fmt.Sprintf("( %s ) > cmd.stdout 2> cmd.stderr", c.Cmd)}
	} else {
		args = []string{"-c", fmt.Sprintf("( %s ) > /${MESOS_SANDBOX}/cmd.stdout 2> /${MESOS_SANDBOX}/cmd.stderr", c.Cmd)}
	}

	return &mesos.CommandInfo{
		Shell:     &shell,
		Value:     &value,
		Arguments: args,
		Uris:      c.Uris,
	}
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
