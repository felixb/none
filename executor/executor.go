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
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"

	executor "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type noneExecutor struct {
	tasksLaunched int
}

func newNoneExecutor() *noneExecutor {
	return &noneExecutor{tasksLaunched: 0}
}

func (executor *noneExecutor) Registered(driver executor.ExecutorDriver, executorInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (executor *noneExecutor) Reregistered(driver executor.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (executor *noneExecutor) Disconnected(executor.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (executor *noneExecutor) runCommand(driver executor.ExecutorDriver, command string) error {
	fmt.Printf("running command: %s", command)

	cmd := exec.Command("sh", "-c", command)
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		fmt.Printf("error starting command: %q", err)
		return err
	} else {
		fmt.Printf("command execution finished: %b", cmd.ProcessState.Success())
	}

	// write stdout + stderr to sandbox
	os.Stdout.Write(stdOut.Bytes())
	os.Stderr.Write(stdErr.Bytes())

	// send stdout + stderr to framework
	driver.SendFrameworkMessage(fmt.Sprintf("stdout:%s", stdOut.String()))
	driver.SendFrameworkMessage(fmt.Sprintf("stderr:%s", stdErr.String()))
	return nil
}

func (executor *noneExecutor) LaunchTask(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	command := taskInfo.Executor.Command.GetArguments()[0]
	fmt.Println("Launching task", taskInfo.GetName(), "with command:", command)
	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}

	executor.tasksLaunched++
	fmt.Println("Total tasks launched ", executor.tasksLaunched)

	err = executor.runCommand(driver, command)
	if err != nil {
		fmt.Println("Task error", taskInfo.GetName())
		finStatus := &mesos.TaskStatus{
			TaskId: taskInfo.GetTaskId(),
			State:  mesos.TaskState_TASK_ERROR.Enum(),
		}
		_, err = driver.SendStatusUpdate(finStatus)
		if err != nil {
			fmt.Println("Got error", err)
		}
	} else {
		// finish task
		fmt.Println("Finishing task", taskInfo.GetName())
		finStatus := &mesos.TaskStatus{
			TaskId: taskInfo.GetTaskId(),
			State:  mesos.TaskState_TASK_FINISHED.Enum(),
		}
		_, err = driver.SendStatusUpdate(finStatus)
		if err != nil {
			fmt.Println("Got error", err)
		}
		fmt.Println("Task finished", taskInfo.GetName())
	}
}

func (executor *noneExecutor) KillTask(executor.ExecutorDriver, *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (executor *noneExecutor) FrameworkMessage(driver executor.ExecutorDriver, msg string) {}

func (executor *noneExecutor) Shutdown(executor.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
}

func (executor *noneExecutor) Error(driver executor.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

// -------------------------- func inits () ----------------- //
func init() {
	flag.Parse()
}

func main() {
	dconfig := executor.DriverConfig{
		Executor: newNoneExecutor(),
	}
	driver, err := executor.NewMesosExecutorDriver(dconfig)

	if err != nil {
		fmt.Println("Unable to create a ExecutorDriver ", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		fmt.Println("Got error:", err)
		return
	}
	driver.Join()
}
