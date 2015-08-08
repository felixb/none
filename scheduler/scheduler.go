package main

import (
	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

type NoneScheduler struct {
	queue         CommandQueuer
	handler       *CommandHandler
	filter        *ResourceFilter
	tasksLaunched int
	tasksFinished int
	tasksFailed   int
	totalTasks    int
}

func NewNoneScheduler(cmdq CommandQueuer, handler *CommandHandler, filter *ResourceFilter) *NoneScheduler {
	return &NoneScheduler{
		queue:   cmdq,
		handler: handler,
		filter:  filter,
	}
}

func (sched *NoneScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Registered with Master", masterInfo)
	sched.handler.SetFrameworkId(frameworkId)
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
		if !sched.filter.FilterOffer(offer) {
			// skip offer if it does not match constraints
			continue
		}

		remainingCpus := SumScalarResources(sched.filter.FilterResources(offer, "cpus"))
		remainingMems := SumScalarResources(sched.filter.FilterResources(offer, "mem"))

		log.Infoln("Received Offer <", offer.Id.GetValue(), "> with cpus=", remainingCpus, " mem=", remainingMems)

		// try to schedule as may tasks as possible for this single offer
		var tasks []*mesos.TaskInfo
		for sched.queue.GetCommand() != nil &&
			sched.queue.GetCommand().MatchesResources(remainingCpus, remainingMems) {

			c := sched.queue.GetCommand()
			c.SlaveId = offer.SlaveId.GetValue()
			sched.handler.CommandLaunched(c)
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
		driver.Abort()
	}
	if c.Status.GetState() == status.GetState() {
		// ignore repeated status updates
		return
	}
	c.Status = status

	// send status update to CommandHandler
	if status.GetState() == mesos.TaskState_TASK_RUNNING {
		sched.handler.CommandRunning(c)
	} else if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.handler.CommandEnded(c)
		sched.handler.CommandFinished(c)
	} else if status.GetState() == mesos.TaskState_TASK_FAILED ||
		status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED {
		sched.handler.CommandEnded(c)
		sched.handler.CommandFailed(c)
	}

	// stop if Commands channel was closed and all tasks are finished
	if sched.queue.Closed() && !sched.handler.HasRunningTasks() {
		log.Infoln("All tasks finished, stopping framework.")
		sched.handler.FinishAllCommands()
		driver.Stop(false)
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
		// TODO: add role to resource allocation?
		Resources: c.GetResources(),
		Container: c.ContainerInfo,
	}
	log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

	return task
}
