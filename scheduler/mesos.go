package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ------ master state --- //

type MasterState struct {
	Slaves []*Slave
}

func NewMasterState(r io.Reader) (*MasterState, error) {
	d := json.NewDecoder(r)
	var s MasterState
	if err := d.Decode(&s); err != nil {
		fmt.Printf("error: %s", err)
		return nil, err
	}
	return &s, nil
}

func FetchMasterState(master *string) (*MasterState, error) {
	url := fmt.Sprintf("http://%s/master/state.json", *master)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return NewMasterState(resp.Body)
}

func (s *MasterState) GetSlave(id string) *Slave {
	for _, slv := range s.Slaves {
		if *slv.Id == id {
			return slv
		}
	}
	return nil
}

// ------ slave --- //

type Slave struct {
	Id       *string
	Hostname *string
	Pid      *string
}

func (s *Slave) GetPort() string {
	return strings.Split(*s.Pid, ":")[1]
}

func (s *Slave) GetUrl() string {
	return fmt.Sprintf("http://%s:%s", *s.Hostname, s.GetPort())
}

func (s *Slave) GetStateUrl() string {
	dir := strings.Split(*s.Pid, "@")[0]
	return fmt.Sprintf("%s/%s/state.json", s.GetUrl(), dir)
}

func (s *Slave) GetState() (*SlaveState, error) {
	resp, err := http.Get(s.GetStateUrl())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return NewSlaveState(resp.Body)
}

// ------ slave state --- //

type SlaveState struct {
	Id         *string
	Pid        *string
	Hostname   *string
	Frameworks []*Framework
}

func NewSlaveState(r io.Reader) (*SlaveState, error) {
	d := json.NewDecoder(r)
	var s SlaveState
	if err := d.Decode(&s); err != nil {
		fmt.Printf("error: %s", err)
		return nil, err
	}
	return &s, nil
}

func (s *SlaveState) GetFramework(id string) *Framework {
	for _, f := range s.Frameworks {
		if *f.Id == id {
			return f
		}
	}
	return nil
}

func (s *SlaveState) GetDirectory(frameworkId, taskId string) *string {
	f := s.GetFramework(frameworkId)
	if f == nil {
		return nil
	}

	e := f.GetExecutor(taskId)
	if e == nil {
		return nil
	}

	return e.Directory
}

// ------ framework --- //

type Framework struct {
	Id        *string
	Container *string
	Executors []*Executor
}

func (f *Framework) GetExecutor(id string) *Executor {
	for _, e := range f.Executors {
		if *e.Id == id {
			return e
		}
	}
	return nil
}

// ------ executors --- //

type Executor struct {
	Id        *string
	Directory *string
	Tasks     []*Task
}

// ------ task --- //

type Task struct {
	Id *string
}

// utils

func FetchSlaveDirInfo(master *string, command *Command) (string, string, error) {
	ms, err := FetchMasterState(master)
	if err != nil {
		return "", "", err
	}

	slv := ms.GetSlave(command.SlaveId)
	if slv == nil {
		return "", "", fmt.Errorf("Unable to find slave with id %s", command.SlaveId)
	}
	ss, err := slv.GetState()
	if err != nil {
		return "", "", err
	}

	d := ss.GetDirectory(command.FrameworkId, command.Id)
	if d == nil {
		return "", "", fmt.Errorf("Unable to find directory for framework %s with task %s", command.FrameworkId, command.Id)
	}

	return slv.GetUrl(), *d, nil
}
