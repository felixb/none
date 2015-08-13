package main

import (
	"fmt"
	"os"

	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
)

type Command struct {
	Id            string
	FrameworkId   string
	SlaveId       string
	Cmd           string
	CpuReq        float64
	MemReq        float64
	ContainerInfo *mesos.ContainerInfo
	Uris          []*mesos.CommandInfo_URI
	Status        *mesos.TaskStatus
	StdoutPailer  *Pailer
	StderrPailer  *Pailer
}

func (c *Command) MatchesResources(cpu, mem float64) bool {
	return c.CpuReq <= cpu && c.MemReq <= mem
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

func (c *Command) GetResources() []*mesos.Resource {
	return []*mesos.Resource{
		util.NewScalarResource("cpus", c.CpuReq),
		util.NewScalarResource("mem", c.MemReq),
	}
}

func (c *Command) StartPailers() {
	c.StdoutPailer = c.createAndStartPailer("cmd.stdout", os.Stdout)
	c.StderrPailer = c.createAndStartPailer("cmd.stderr", os.Stderr)
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

func (c *Command) DownloadFile(path *string) error {
	f, err := os.Create(*path)
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := NewDownloader(f, master, c, *path)
	if err != nil {
		return err
	}
	_, err = d.Download()
	return err
}

// private

func (c *Command) createAndStartPailer(file string, w StringWriter) *Pailer {
	p, err := NewPailer(w, master, c, file)
	if err != nil {
		log.Errorf("Unable to start pailer for task %s: %s\n", c.Id, err)
		return nil
	} else {
		p.Start()
		return p
	}
}
