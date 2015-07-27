// +build none-scheduler

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
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	archivex "github.com/jhoonb/archivex"
	"github.com/mesos/mesos-go/auth"
	"github.com/mesos/mesos-go/auth/sasl"
	"github.com/mesos/mesos-go/auth/sasl/mech"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"golang.org/x/net/context"
)

const (
	DEFAULT_CPUS_PER_TASK = 1
	DEFAULT_MEM_PER_TASK  = 128
	DEFAULT_ARTIFACT_PORT = 10080
	DEFAULT_DRIVER_PORT   = 10050
	EXECUTOR_FILENAME     = "none-executor"
	WORKDIR_ARCHIVE       = "workdir.tar.gz"
)

var (
	hostname     = flag.String("hostname", "", "Overwrite hostname")
	ip           = flag.String("ip", "127.0.0.1", "Binding address for framework and artifact server")
	port         = flag.Uint("port", DEFAULT_DRIVER_PORT, "Binding port for framework")
	artifactPort = flag.Int("artifactPort", DEFAULT_ARTIFACT_PORT, "Binding port for artifact server")
	master       = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	authProvider = flag.String("mesos-authentication-provider", sasl.ProviderName,
		fmt.Sprintf("Authentication provider to use, default is SASL that supports mechanisms: %+v", mech.ListSupported()))
	mesosAuthPrincipal  = flag.String("mesos-authentication-principal", "", "Mesos authentication principal.")
	mesosAuthSecretFile = flag.String("mesos-authentication-secret-file", "", "Mesos authentication secret file.")
	user                = flag.String("user", "", "Run task as specified user. Defaults to current user.")
	framworkName        = flag.String("framework-name", "NONE", "Framework name")
	sendWorkdir         = flag.Bool("send-workdir", true, "Send current working dir to executor.")
	cpuPerTask          = flag.Float64("cpuPerTask", DEFAULT_CPUS_PER_TASK, "CPU reservation for task execution")
	memPerTask          = flag.Float64("memPerTask", DEFAULT_MEM_PER_TASK, "Memory resveration for task execution")
	command             = flag.String("command", "", "Command to run on the cluster")
)

type NoneScheduler struct {
	executor      *mesos.ExecutorInfo
	command       *mesos.CommandInfo
	cpuPerTask    float64
	memPerTask    float64
	tasksLaunched int
	tasksFinished int
	totalTasks    int
}

func newNoneScheduler(exec *mesos.ExecutorInfo, cpus, mem float64) *NoneScheduler {
	return &NoneScheduler{
		executor:      exec,
		cpuPerTask:    cpus,
		memPerTask:    mem,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    1,
	}
}

func (sched *NoneScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Registered with Master ", masterInfo)
}

func (sched *NoneScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Re-Registered with Master ", masterInfo)
}

func (sched *NoneScheduler) Disconnected(sched.SchedulerDriver) {}

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

		var tasks []*mesos.TaskInfo
		for sched.tasksLaunched < sched.totalTasks &&
			sched.cpuPerTask <= remainingCpus &&
			sched.memPerTask <= remainingMems {

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}

			task := &mesos.TaskInfo{
				Name:     proto.String("none-task-" + taskId.GetValue()),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", sched.cpuPerTask),
					util.NewScalarResource("mem", sched.memPerTask),
				},
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			remainingCpus -= sched.cpuPerTask
			remainingMems -= sched.memPerTask
		}
		log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *NoneScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
	if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.tasksFinished++
	}

	if sched.tasksFinished >= sched.totalTasks {
		log.Infoln("Total tasks completed, stopping framework.")
		driver.Stop(false)
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Infoln(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

func (sched *NoneScheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {}

func (sched *NoneScheduler) FrameworkMessage(driver sched.SchedulerDriver, exec *mesos.ExecutorID, slave *mesos.SlaveID, message string) {
	parts := strings.SplitN(message, ":", 2)
	if parts[0] == "stdout" {
		os.Stdout.WriteString(parts[1])
	} else if parts[0] == "stderr" {
		os.Stderr.WriteString(parts[1])
	} else {
		log.Warningln("Dropping invalid message")
	}
}

func (sched *NoneScheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {}
func (sched *NoneScheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
}

func (sched *NoneScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}

// ----------------------- func init() ------------------------- //

func init() {
	flag.Parse()
	log.Infoln("Initializing the None Scheduler...")
}

// returns (downloadURI, basename(path))
func serveArtifact(path, base string) (*string, string) {
	serveFile := func(pattern string, filename string) {
		http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filename)
		})
	}

	serveFile("/"+base, path)

	hostURI := fmt.Sprintf("http://%s:%d/%s", *ip, *artifactPort, base)
	log.Infof("Hosting artifact '%s' at '%s'", path, hostURI)

	return &hostURI, base
}

func tarWorkdir() (*string, error) {
	path := fmt.Sprintf("%s/none-workdir-%d.tar.gz", os.TempDir(), os.Getpid())
	tar := new(archivex.TarFile)
	tar.Create(path)
	tar.AddAll(".", true)
	tar.Close()
	return &path, nil
}

func prepareExecutorInfo(workdirPath *string) *mesos.ExecutorInfo {
	executorPath := filepath.Join(filepath.Dir(os.Args[0]), EXECUTOR_FILENAME)
	executorUris := []*mesos.CommandInfo_URI{}
	uri, executorCmd := serveArtifact(executorPath, filepath.Base(executorPath))
	executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(true)})

	if workdirPath != nil {
		uri, _ := serveArtifact(*workdirPath, WORKDIR_ARCHIVE)
		executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(false)})
	}

	executorCommand := fmt.Sprintf("./%s", executorCmd)

	go http.ListenAndServe(fmt.Sprintf("%s:%d", *ip, *artifactPort), nil)
	log.Infoln("Serving executor artifacts...")

	shell := false
	args := []string{*command}

	// Create mesos scheduler driver.
	return &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("NONE Executor"),
		Source:     proto.String("none_executor"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(executorCommand),
			Uris:      executorUris,
			Arguments: args,
			Shell:     &shell,
		},
	}
}

func parseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}

// ----------------------- func main() ------------------------- //

func main() {

	// tar workdir and make sure it gets deleted
	var workdirPath *string
	var err error
	if *sendWorkdir {
		workdirPath, err = tarWorkdir()
		if err != nil {
			log.Errorln("error creating workdir tar:", err)
		}
		defer os.Remove(*workdirPath)
	}

	// build command executor
	exec := prepareExecutorInfo(workdirPath)

	// the framework
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(*user),
		Name: proto.String("NONE"),
	}

	// credentials
	cred := (*mesos.Credential)(nil)
	if *mesosAuthPrincipal != "" {
		fwinfo.Principal = proto.String(*mesosAuthPrincipal)
		secret, err := ioutil.ReadFile(*mesosAuthSecretFile)
		if err != nil {
			log.Fatal(err)
		}
		cred = &mesos.Credential{
			Principal: proto.String(*mesosAuthPrincipal),
			Secret:    secret,
		}
	}
	bindingAddress := parseIP(*ip)
	scheduler := newNoneScheduler(exec, *cpuPerTask, *memPerTask)
	config := sched.DriverConfig{
		Scheduler:        scheduler,
		Framework:        fwinfo,
		Master:           *master,
		Credential:       cred,
		HostnameOverride: *hostname,
		BindingAddress:   bindingAddress,
		BindingPort:      uint16(*port),
		WithAuthContext: func(ctx context.Context) context.Context {
			ctx = auth.WithLoginProvider(ctx, *authProvider)
			ctx = sasl.WithBindingAddress(ctx, bindingAddress)
			return ctx
		},
	}
	driver, err := sched.NewMesosSchedulerDriver(config)

	if err != nil {
		log.Errorln("Unable to create a SchedulerDriver ", err.Error())
	}

	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}

}
