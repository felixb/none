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
	"os"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

type NoneScheduler struct {
	C             chan *Command
	nextCommand   *Command
	commands      map[string]*Command
	uris          []*mesos.CommandInfo_URI
	frameworkId   *mesos.FrameworkID
	cpuPerTask    float64
	memPerTask    float64
	tasksLaunched int
	tasksFinished int
	totalTasks    int
	running       bool
}

func NewNoneScheduler(uris []*mesos.CommandInfo_URI, cpus, mem float64) *NoneScheduler {
	return &NoneScheduler{
		C:             make(chan *Command, 10),
		nextCommand:   nil,
		commands:      make(map[string]*Command, 10),
		uris:          uris,
		frameworkId:   nil,
		cpuPerTask:    cpus,
		memPerTask:    mem,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    0,
		running:       true,
	}
}

func (sched *NoneScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Registered with Master", masterInfo)
	sched.frameworkId = frameworkId
}

func (sched *NoneScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {}

func (sched *NoneScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Framework Disconnected")
}

// process incoming offers and try to schedule new tasks as they come in on the channel
func (sched *NoneScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	for _, offer := range offers {
		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}

		log.Infoln("Received Offer <", offer.Id.GetValue(), "> with cpus=", cpus, " mem=", mems)

		remainingCpus := cpus
		remainingMems := mems

		if sched.nextCommand == nil {
			sched.fetchNextCommand()
		}

		// try to schedule as may tasks as possible for this single offer
		var tasks []*mesos.TaskInfo
		for sched.nextCommand != nil &&
			sched.cpuPerTask <= remainingCpus &&
			sched.memPerTask <= remainingMems {

			sched.tasksLaunched++

			tId := strconv.Itoa(sched.tasksLaunched)
			sched.nextCommand.Id = tId
			taskId := &mesos.TaskID{
				Value: proto.String(tId),
			}

			shell := true

			task := &mesos.TaskInfo{
				Name:    proto.String("none-task-" + taskId.GetValue()),
				TaskId:  taskId,
				SlaveId: offer.SlaveId,
				Command: &mesos.CommandInfo{
					Shell: &shell,
					Value: &sched.nextCommand.Cmd,
					Uris:  sched.uris,
				},
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", sched.cpuPerTask),
					util.NewScalarResource("mem", sched.memPerTask),
				},
				Data: []byte(sched.nextCommand.Cmd),
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			sched.commands[tId] = sched.nextCommand

			remainingCpus -= sched.cpuPerTask
			remainingMems -= sched.memPerTask
			sched.fetchNextCommand()
		}
		log.Infoln("Launching", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *NoneScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), "is in state", status.State.Enum().String())

	c := sched.commands[status.GetTaskId().GetValue()]
	if c == nil {
		log.Errorf("Unable to find command for task %s", status.GetTaskId().GetValue())
	} else {
		c.Status = status

		if status.GetState() == mesos.TaskState_TASK_RUNNING {
			c.StdoutPailer = sched.createAndStartPailer(driver, status, "stdout", os.Stdout)
			c.StderrPailer = sched.createAndStartPailer(driver, status, "stderr", os.Stderr)
		}
	}

	if status.GetState() == mesos.TaskState_TASK_FINISHED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		sched.tasksFinished++

		if c != nil {
			c.StopPailers()
		}
	}

	// stop if Commands channel was closed and all tasks have finished
	if !sched.running && sched.tasksFinished >= sched.totalTasks {
		log.Infoln("Total tasks completed, stopping framework.")
		for _, c := range sched.commands {
			c.WaitForPailers()
		}
		driver.Stop(false)
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED {
		log.Infoln(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

func (sched *NoneScheduler) OfferRescinded(driver sched.SchedulerDriver, offer *mesos.OfferID) {
	log.Infoln("Rescined offer", *offer)
}

func (sched *NoneScheduler) FrameworkMessage(driver sched.SchedulerDriver, exec *mesos.ExecutorID, slave *mesos.SlaveID, message string) {
	log.Infof("Framework message: %s", message)
}

func (sched *NoneScheduler) SlaveLost(driver sched.SchedulerDriver, offer *mesos.SlaveID) {
	log.Infoln("Lost slave", *offer)
}
func (sched *NoneScheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
}

func (sched *NoneScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}

// private

func (sched *NoneScheduler) fetchNextCommand() {
	select {
	case sched.nextCommand = <-sched.C:
		if sched.nextCommand != nil {
			sched.totalTasks++
			log.Infoln("Schedule next command from queue:", strings.TrimSpace(sched.nextCommand.Cmd))
		} else {
			// channel was closed, stop listening for new commands
			sched.running = false
		}
	default:
		sched.nextCommand = nil
	}
}

func printer(f *os.File, out chan string) {
	for {
		f.WriteString(<-out)
	}
}

func (sched *NoneScheduler) createAndStartPailer(driver sched.SchedulerDriver, status *mesos.TaskStatus, file string, w *os.File) *Pailer {
	p, err := NewPailer(master, status.GetSlaveId(), sched.frameworkId, status.GetTaskId(), file)
	if err != nil {
		log.Errorf("Unable to start pailer for task %s: %s\n", status.GetTaskId().GetValue(), err)
		return nil
	} else {
		p.Start()
		go printer(w, p.C)
		return p
	}
}
