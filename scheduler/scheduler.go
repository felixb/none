package main

import (
	"os"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

type NoneScheduler struct {
	queue         CommandQueuer
	frameworkId   *mesos.FrameworkID
	constraints   Constraints
	tasksLaunched int
	tasksFinished int
	tasksFailed   int
	totalTasks    int
	running       bool
}

func NewNoneScheduler(cmdq CommandQueuer, constraints Constraints) *NoneScheduler {
	return &NoneScheduler{
		queue:         cmdq,
		frameworkId:   nil,
		constraints:   constraints,
		tasksLaunched: 0,
		tasksFinished: 0,
		tasksFailed:   0,
		totalTasks:    0,
		running:       true,
	}
}

func (sched *NoneScheduler) HasFailures() bool {
	return sched.tasksFailed > 0
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
	if sched.queue.Next() == nil {
		// no command to launch, we don't need to parse anything
		return
	}

	for _, offer := range offers {
		// match constraints
		if !sched.constraints.Match(offer) {
			// skip offer if it does not match constraints
			continue
		}

		// check resources
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

		// try to schedule as may tasks as possible for this single offer
		var tasks []*mesos.TaskInfo
		for sched.queue.GetCommand() != nil &&
			sched.queue.GetCommand().CpuReq <= remainingCpus &&
			sched.queue.GetCommand().MemReq <= remainingMems {

			c := sched.queue.GetCommand()
			task := sched.prepareTaskInfo(offer, c)
			tasks = append(tasks, task)

			remainingCpus -= c.CpuReq
			remainingMems -= c.MemReq
			sched.queue.Next()
		}
		log.Infoln("Launching", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *NoneScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	taskId := status.GetTaskId().GetValue()
	log.Infoln("Status update: task", taskId, "is in state", status.State.Enum().String())

	c := sched.queue.GetCommandById(taskId)
	if c == nil {
		log.Errorln("Unable to find command for task", taskId)
	} else {
		c.Status = status

		if status.GetState() == mesos.TaskState_TASK_RUNNING {
			c.StdoutPailer = sched.createAndStartPailer(status, "cmd.stdout", os.Stdout)
			c.StderrPailer = sched.createAndStartPailer(status, "cmd.stderr", os.Stderr)
		}
	}

	if status.GetState() == mesos.TaskState_TASK_FINISHED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		sched.tasksFinished++

		if status.GetState() == mesos.TaskState_TASK_FAILED {
			sched.tasksFailed++
		}

		if c != nil {
			c.StopPailers()
		}
	}

	// stop if Commands channel was closed and all tasks have finished
	if !sched.running && sched.tasksFinished >= sched.totalTasks {
		log.Infoln("Total tasks completed, stopping framework.")
		for _, c := range sched.queue.GetCommands() {
			c.WaitForPailers()
		}
		driver.Stop(false)
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED {
		sched.tasksFailed++
		log.Infoln(
			"Aborting because task", taskId,
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
	log.Infoln("Framework message:", message)
}

func (sched *NoneScheduler) SlaveLost(driver sched.SchedulerDriver, slave *mesos.SlaveID) {
	log.Infoln("Lost slave", *slave)
}

func (sched *NoneScheduler) ExecutorLost(driver sched.SchedulerDriver, executor *mesos.ExecutorID, slave *mesos.SlaveID, i int) {
	log.Infoln("Lost executor", *executor)
}

func (sched *NoneScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}

// private

func (sched *NoneScheduler) prepareTaskInfo(offer *mesos.Offer, c *Command) *mesos.TaskInfo {
	sched.tasksLaunched++

	task := &mesos.TaskInfo{
		Name:    proto.String("none-task-" + c.Id),
		TaskId:  util.NewTaskID(c.Id),
		SlaveId: offer.SlaveId,
		Command: c.GetCommandInfo(),
		Resources: []*mesos.Resource{
			util.NewScalarResource("cpus", c.CpuReq),
			util.NewScalarResource("mem", c.MemReq),
		},
		Container: c.ContainerInfo,
	}
	log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

	return task
}

func printer(f *os.File, out chan string) {
	for {
		f.WriteString(<-out)
	}
}

func (sched *NoneScheduler) createAndStartPailer(status *mesos.TaskStatus, file string, w *os.File) *Pailer {
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
