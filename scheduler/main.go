package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"

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
	sendWorkdir         = flag.Bool("send-workdir", true, "Send current working dir to executor.")
	cpuPerTask          = flag.Float64("cpu-per-task", DEFAULT_CPUS_PER_TASK, "CPU reservation for task execution")
	memPerTask          = flag.Float64("mem-per-task", DEFAULT_MEM_PER_TASK, "Memory resveration for task execution")
	command             = flag.String("command", "", "Command to run on the cluster")
	containerJson       = flag.String("container", "", "Container definition as JSON, overrules dockerImage")
	dockerImage         = flag.String("docker-image", "", "Docker image for running the commands in")
	constraints         = flag.String("constraints", "", "Constraints for selecting mesos slaves, format: 'attribute:operant[:value][;..]'")
	version             = flag.Bool("version", false, "Show NONE version.")
)

// parse command line flags
func init() {
	flag.Parse()
	log.Infoln("Initializing the None Scheduler...")
	// each pailer is generating 2 threads which is waiting most of the time
	numThreads := runtime.NumCPU()*2 + 1
	log.Infof("Setting max number of threads to %d", numThreads)
	runtime.GOMAXPROCS(numThreads)
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

// tar workdir
// returns path to local artifact
func tarWorkdir() *string {
	if !*sendWorkdir {
		return nil
	}

	path := fmt.Sprintf("%s/none-workdir-%d.tar.gz", os.TempDir(), os.Getpid())
	tar := new(archivex.TarFile)
	tar.Create(path)
	tar.AddAll(".", true)
	tar.Close()
	return &path
}

// server workdir and executor artifacts
// returns (executor command, artifact uris)
func exportArtifacts(workdirPath *string) []*mesos.CommandInfo_URI {
	executorUris := []*mesos.CommandInfo_URI{}
	if workdirPath != nil {
		uri := serveArtifact(*workdirPath, WORKDIR_ARCHIVE)
		executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(false)})
	}

	go http.ListenAndServe(fmt.Sprintf("%s:%d", *address, *artifactPort), nil)
	log.Infoln("Serving executor artifacts...")

	return executorUris
}

// create the framework data structure
func prepareFrameworkInfo() *mesos.FrameworkInfo {
	return &mesos.FrameworkInfo{
		User: proto.String(*user),
		Name: proto.String(*framworkName),
	}
}

// create the executor data structure
func prepareExecutorInfo(executorCommand string, executorUris []*mesos.CommandInfo_URI) *mesos.ExecutorInfo {
	shell := false

	// Create mesos scheduler driver.
	return &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("NONE Executor"),
		Source:     proto.String("none/executor"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(executorCommand),
			Uris:      executorUris,
			Shell:     &shell,
			Arguments: []string{""},
		},
	}
}

// create credentials data structure
func prepateCredentials(fwinfo *mesos.FrameworkInfo) *mesos.Credential {
	if *mesosAuthPrincipal != "" {
		fwinfo.Principal = proto.String(*mesosAuthPrincipal)
		secret, err := ioutil.ReadFile(*mesosAuthSecretFile)
		if err != nil {
			log.Fatal(err)
		}
		return &mesos.Credential{
			Principal: proto.String(*mesosAuthPrincipal),
			Secret:    secret,
		}
	} else {
		return nil
	}
}

func prepareContainer() *mesos.ContainerInfo {
	if containerJson != nil && *containerJson != "" {
		var ci mesos.ContainerInfo
		if err := json.Unmarshal([]byte(*containerJson), &ci); err != nil {
			log.Fatalf("Unable to parse container info: %s", err)
		}
		return &ci
	} else if dockerImage != nil && *dockerImage != "" {
		return &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_DOCKER.Enum(),
			Docker: &mesos.ContainerInfo_DockerInfo{
				Image: dockerImage,
			},
		}
	}
	return nil
}

// create the driver data structure
func prepareDriver(scheduler *NoneScheduler, fwinfo *mesos.FrameworkInfo, cred *mesos.Credential) sched.DriverConfig {
	bindingAddress := parseIP(*address)
	return sched.DriverConfig{
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

func startCommand(scheduler *NoneScheduler, cmd *string) {
	scheduler.C <- &Command{
		Cmd: *cmd,
	}
}

func startcommands(scheduler *NoneScheduler) {
	reader := bufio.NewReader(os.Stdin)
	cmd, err := reader.ReadString('\n')
	for err == nil {
		startCommand(scheduler, &cmd)
		cmd, err = reader.ReadString('\n')
	}
	close(scheduler.C)
}

// ----------------------- func main() ------------------------- //

func main() {
	if *version {
		fmt.Printf("NONE v%s\n", VERSION)
		os.Exit(0)
	}

	workdirPath := tarWorkdir()
	if workdirPath != nil {
		defer os.Remove(*workdirPath)
	}
	fwinfo := prepareFrameworkInfo()
	cred := prepateCredentials(fwinfo)
	cs, err := ParseConstraints(constraints)
	if err != nil {
		log.Errorf("Error parsing constraints: %s", err)
	}
	scheduler := NewNoneScheduler(prepareContainer(), cs, exportArtifacts(workdirPath), *cpuPerTask, *memPerTask)
	config := prepareDriver(scheduler, fwinfo, cred)

	driver, err := sched.NewMesosSchedulerDriver(config)
	if err != nil {
		log.Errorln("Unable to create a SchedulerDriver ", err.Error())
	}

	if command != nil && *command != "" {
		// queue single command for execution
		startCommand(scheduler, command)
		close(scheduler.C)
	} else {
		// queue commands from stdin for execution
		// non-blocking
		go startcommands(scheduler)
	}

	// run the driver and wait for it to finish
	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}

	if scheduler.HasFailures() {
		os.Exit(1)
	}
}
