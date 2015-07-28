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
	"bufio"
	"flag"
	"fmt"
	"io"
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

func (executor *noneExecutor) LaunchTask(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	command := string(taskInfo.Data)
	fmt.Println("Launching task", taskInfo.GetName(), "with command:", command)
	go executor.launchTask(driver, taskInfo, command)
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

// -------------------------- private ----------------- //

func (executor *noneExecutor) sendStatus(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo, state *mesos.TaskState) {
	taskStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  state,
	}
	_, err := driver.SendStatusUpdate(taskStatus)
	if err != nil {
		fmt.Println("Error sending status update", err)
	}
}

func (executor *noneExecutor) forwardOutput(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo, kind string, w io.Writer, r io.ReadCloser) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		b := append(scanner.Bytes(), '\n')
		w.Write(b)
		driver.SendFrameworkMessage(fmt.Sprintf("%s:%s:%s\n", taskInfo.TaskId.GetValue(), kind, scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading %s: %s", kind, err)
	}
}

// run a command
func (executor *noneExecutor) runCommand(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo, command string) error {
	fmt.Printf("running command: %s", command)

	cmd := exec.Command("sh", "-c", command)

	// connect pipes to stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("error reading stdout: %q", err)
		return err
	}
	go executor.forwardOutput(driver, taskInfo, "stdout", os.Stdout, stdout)

	// connect pipes to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("error reading stderr: %q", err)
		return err
	}
	go executor.forwardOutput(driver, taskInfo, "stderr", os.Stderr, stderr)

	// run the command
	return cmd.Run()
}

func (executor *noneExecutor) launchTask(driver executor.ExecutorDriver, taskInfo *mesos.TaskInfo, command string) {
	executor.sendStatus(driver, taskInfo, mesos.TaskState_TASK_RUNNING.Enum())

	executor.tasksLaunched++
	fmt.Println("Total tasks launched ", executor.tasksLaunched)

	err := executor.runCommand(driver, taskInfo, command)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// the command has exited with an exit code != 0
			fmt.Println("Failed task", taskInfo.GetName())
			executor.sendStatus(driver, taskInfo, mesos.TaskState_TASK_FAILED.Enum())
		} else {
			// something bad happend
			fmt.Println("Task error", taskInfo.GetName())
			executor.sendStatus(driver, taskInfo, mesos.TaskState_TASK_ERROR.Enum())
		}
	} else {
		// finished task
		fmt.Println("Finished task", taskInfo.GetName())
		executor.sendStatus(driver, taskInfo, mesos.TaskState_TASK_FINISHED.Enum())
	}
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
