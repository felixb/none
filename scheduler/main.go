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
	defaultHostname, _ = os.Hostname()
	hostname           = flag.String("hostname", "", "Overwrite hostname")
	address            = flag.String("address", defaultHostname, "Binding address for framework and artifact server")
	port               = flag.Uint("port", DEFAULT_DRIVER_PORT, "Binding port for framework")
	artifactPort       = flag.Int("artifactPort", DEFAULT_ARTIFACT_PORT, "Binding port for artifact server")
	master             = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	authProvider       = flag.String("mesos-authentication-provider", sasl.ProviderName,
		fmt.Sprintf("Authentication provider to use, default is SASL that supports mechanisms: %+v", mech.ListSupported()))
	mesosAuthPrincipal  = flag.String("mesos-authentication-principal", "", "Mesos authentication principal.")
	mesosAuthSecretFile = flag.String("mesos-authentication-secret-file", "", "Mesos authentication secret file.")
	user                = flag.String("user", "", "Run task as specified user. Defaults to current user.")
	framworkName        = flag.String("framework-name", "NONE", "Framework name")
	executorPath        = flag.String("executor", filepath.Join(filepath.Dir(os.Args[0]), EXECUTOR_FILENAME), "Executor binary")
	sendWorkdir         = flag.Bool("send-workdir", true, "Send current working dir to executor.")
	cpuPerTask          = flag.Float64("cpuPerTask", DEFAULT_CPUS_PER_TASK, "CPU reservation for task execution")
	memPerTask          = flag.Float64("memPerTask", DEFAULT_MEM_PER_TASK, "Memory resveration for task execution")
	command             = flag.String("command", "", "Command to run on the cluster")
)

// parse command line flags
func init() {
	flag.Parse()
	log.Infoln("Initializing the None Scheduler...")
}

// returns uri pointing to artifact
func serveArtifact(path, base string) *string {
	serveFile := func(pattern string, filename string) {
		http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filename)
		})
	}

	serveFile("/"+base, path)

	hostURI := fmt.Sprintf("http://%s:%d/%s", *address, *artifactPort, base)
	log.Infof("Hosting artifact '%s' at '%s'", path, hostURI)

	return &hostURI
}

// tar current workdir
// returns path to local artifact
func tarWorkdir() (*string, error) {
	path := fmt.Sprintf("%s/none-workdir-%d.tar.gz", os.TempDir(), os.Getpid())
	tar := new(archivex.TarFile)
	tar.Create(path)
	tar.AddAll(".", true)
	tar.Close()
	return &path, nil
}

// server workdir and executor artifacts
// returns (executor command, artifact uris)
func exportArtifacts(workdirPath *string) (string, []*mesos.CommandInfo_URI) {
	executorUris := []*mesos.CommandInfo_URI{}
	executorCmd := filepath.Base(*executorPath)
	uri := serveArtifact(*executorPath, executorCmd)
	executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(true)})

	if workdirPath != nil {
		uri := serveArtifact(*workdirPath, WORKDIR_ARCHIVE)
		executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(false)})
	}

	executorCommand := fmt.Sprintf("./%s", executorCmd)

	go http.ListenAndServe(fmt.Sprintf("%s:%d", *address, *artifactPort), nil)
	log.Infoln("Serving executor artifacts...")

	return executorCommand, executorUris
}

// create the executor
func prepareExecutorInfo(executorCommand string, executorUris []*mesos.CommandInfo_URI) *mesos.ExecutorInfo {
	shell := false
	args := []string{*command}

	// Create mesos scheduler driver.
	return &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("NONE Executor"),
		Source:     proto.String("none/executor"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(executorCommand),
			Uris:      executorUris,
			Arguments: args,
			Shell:     &shell,
		},
	}
}

// resolve hostname to ip
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

	// tar workdir and make sure it gets deleted after execution
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
	exec := prepareExecutorInfo(exportArtifacts(workdirPath))

	// the framework
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(*user),
		Name: proto.String(*framworkName),
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

	// create the scheduler
	scheduler := NewNoneScheduler(exec, *cpuPerTask, *memPerTask)

	// configure the driver
	bindingAddress := parseIP(*address)
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

	// run the driver and wait for it to finish
	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}

}
