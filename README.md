# NONE - aNy cOmmand oN mEsos

[![Build Status](https://travis-ci.org/felixb/none.svg)](https://travis-ci.org/felixb/none)

With NONE, you can run any abitrary command on a mesos cluster. With or with out docker.

## Run it

Schedule any command by running the binary.
The current working directory is transfered to the slave before execution.
Stdout and Stderr is forwarded to your terminal.

Example for running a single command:

    $ ./none-scheduler -address=10.141.141.1  -master=10.141.141.10:5050  -command 'echo woooohoooooooo'
    woooohoooooooo

Example for running three commands in parallel:

    $ ./none-scheduler -address=10.141.141.1  -master=10.141.141.10:5050 2>/dev/null <<EOF
    > echo task 1; date
    > echo task 2; ls -l
    > echo task 3; uname -a
    > EOF
    task 2
    total 21180
    drwxr-xr-x 2 flx flx     4096 Jul 31 18:06 fixtures
    drwxr-xr-x 7 flx flx     4096 Jul 31 18:06 git
    -rw-rw-r-- 1 flx flx    11323 Jul 27 11:28 LICENSE
    -rw-rw-r-- 1 flx flx      286 Jul 31 15:52 Makefile
    -rwxrwxr-x 1 flx flx 10621272 Jul 31 18:00 none-scheduler
    -rw-rw-rw- 1 flx flx      233 Jul 31 15:45 NOTICE
    -rw-rw-rw- 1 flx flx     2372 Jul 31 17:53 README.md
    drwxr-xr-x 2 flx flx     4096 Jul 31 18:06 scheduler
    -rw-r--r-- 1 flx flx     1079 Jul 31 18:06 stderr
    -rw-r--r-- 1 flx flx       88 Jul 31 18:06 stdout
    -rw-r--r-- 1 flx flx 11018240 Jul 31 18:06 workdir.tar.gz
    task 1
    Fri Jul 31 18:06:18 UTC 2015
    task 3
    Linux mesos 3.16.0-45-generic #60~14.04.1-Ubuntu SMP Fri Jul 24 21:16:23 UTC 2015 x86_64 x86_64 x86_64 GNU/Linux

### Full list of available command line options

#### Communication

 * `-address="your-hostname"`: Binding address for framework and artifact server
 * `-artifactPort=10080`: Binding port for artifact server
 * `-hostname=""`: Overwrite hostname
 * `-master="127.0.0.1:5050"`: Master address <ip:port>
 * `-port=10050`: Binding port for framework

#### Framework

 * `-framework-name="NONE"`: Framework name
 * `-decode-routines=1`: Number of decoding routines
 * `-encode-routines=1`: Number of encoding routines
 * `-send-routines=1`: Number of network sending routines

#### Tasks

 * `-command=""`: Command to run on the cluster
 * `-constraints=""`: Constraints for selecting mesos slaves, format: `attribute:operant[:value][;..]`
 * `-container=""`: Container definition as JSON, overrules dockerImage
 * `-cpu-per-task=1`: CPU reservation for task execution
 * `-docker-image=""`: Docker image for running the commands in
 * `-mem-per-task=128`: Memory resveration for task execution
 * `-user=""`: Run task as specified user. Defaults to current user.
 * `-send-workdir=true`: Send current working dir to executor.

#### Authentication

 * `-mesos-authentication-principal=""`: Mesos authentication principal.
 * `-mesos-authentication-provider="SASL"`: Authentication provider to use, default is SASL that supports mechanisms: [CRAM-MD5]
 * `-mesos-authentication-secret-file=""`: Mesos authentication secret file.

#### Logging

 * `-alsologtostderr=false`: log to standard error as well as files
 * `-log_backtrace_at=:0`: when logging hits line file:N, emit a stack trace
 * `-log_dir=""`: If non-empty, write log files in this directory
 * `-logtostderr=false`: log to standard error instead of files
 * `-stderrthreshold=0`: logs at or above this threshold go to stderr
 * `-v=0`: log level for V logs
 * `-vmodule=`: comma-separated list of pattern=N settings for file-filtered logging

### Constraints

You may apply constraints for selecting mesos slaves by attributes with the `-constraints` flag.

The following constraints are supported:

* `EQUALS`: selecting only slaves with an attribute with exactly the specified value. `foo:EQUALS:bar` requires a slave with the attribute `foo` and value `bar`.

## Build it

Build NONE with make by running:

 * `make get-deps`
 * `make build`

## Contribute

Please fork and send a PR.

## Licensing

NONE is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/felixb/none/blob/master/LICENSE) for the full
license text.