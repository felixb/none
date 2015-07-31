/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
