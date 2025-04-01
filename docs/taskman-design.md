---
authors: Michael Wurtz (mikewurtz1@gmail.com)
state: draft
---

# RFD - Arbitrary Linux Process Manager

## What

This RFD proposes the design and implementation of a CLI, gRPC server, and reusable library that allow users/systems to run arbitrary linux processes.

## Why
Users and systems often need the ability to run arbitrary Linux processes remotely — whether for automation, job execution, remote debugging, or platform orchestration. However, doing this in a secure and user-friendly way is non-trivial.

This implementation provides a clean, modular solution for that problem by exposing a gRPC-based API to start and manage processes on a remote server. Users can interact with this API through a command-line interface (CLI), making the system easy to use for both people and automation workflows (e.g., CI/CD pipelines, job queues, developer tools).

## Details

#### Required Approvers

```
# Required Approvers (two required)
* Engineering:  @tigrato || @timothyb89 || @russjones || @zmb3 || @rosstimothy || @r0mant
```

### UX

Users will interact with the exposed gRPC server APIs through commands provided by the `taskman` command line tool. This tool will utilize a gRPC client to communicate with the gRPC server. The user must pass a `--user-id` flag that allows the user to select which client they are. The CLI will pick from a hardcoded set of certificates in the application that will be part of the gRPC requests in order to perform authentication. The following CLI commands will be provided in the taskman client:

#### start
The start command starts a new task given the passed in full command and associates the task with the provided user-id. If successful the generated task-id will be returned.

```
$ taskman start --help
Usage:
  taskman start "<command>" --user-id <user-id> [--help]

Description:
  Start a new task by executing the specified command. The command must be provided in quotes, 
  and the --user-id flag is required to identify the client initiating the request.

Arguments:
  <command>
        The full command to execute, including the executable path.
        Example: "/bin/ls"

Options:
  --user-id <user-id>
  		The user or client ID issuing the request (e.g., client001). This flag is required.
  --help
  		Display help information for the start command.

Example:
  $ taskman start "/bin/ls /myFolder" --user-id client001

```
Example Output

```
$ taskman start "/bin/ls /myFolder" --user-id client001
TASK ID
-------
a7da14c7-b47a-4535-a263-5bb26e503002

```

#### get-status
The get-status command is how the user can query an existing task by task-id. The task must have been created by the user in order for the server to return the task details. If the task does not exist or is not associated with the user a `NOT_FOUND` will be returned.

```
$ taskman get-status --help
Usage:
  taskman get-status <task-id> --user-id <user-id> [--help]

Description:
  Retrieve the status of a task using its unique task ID. The command displays details such as whether 
  the task is running, start time, process id, and exit code and end time if the process has ended.

Arguments:
  <task-id>             The UUID of the task to query (e.g., a7da14c7-b47a-4535-a263-5bb26e503002)

Options:
  --user-id <user-id>   
  		The user or client ID issuing the request (e.g., client001). This flag is required.
  --help
  		Display help information for the get-status command.

Example:
  $ taskman get-status a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001

```
Example output

```
$ taskman get-status a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001
TASK ID                                START TIME           PID   RUNNING  EXIT CODE  END TIME
-------                                ----------           ---   -------  ---------  ---------
a7da14c7-b47a-4535-a263-5bb26e503002   2024-11-10 23:00:00  1234  false    0          2024-11-10 23:05:00
```

#### stop
The stop command will stop the provided task id if the task is running and owned by the user. If the task is not running this command will have no effect. If the user does not own the task or the task does not exist a `NOT_FOUND` will be returned.

```
Usage:
  taskman stop <task-id> --user-id <user-id> [--help]

Description:
  Stop a running task identified by its unique task ID. This command terminates the specified task 
  and confirms the request was accepted.

Arguments:
  <task-id>
        The unique identifier (UUID) of the task to stop.
        Example: a7da14c7-b47a-4535-a263-5bb26e503002

Options:
  --user-id <user-id>
        The user or client ID issuing the request (e.g., client001). This flag is required.
  --help
        Display help information for the stop command.

Example:
  $ taskman stop a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001

```

Example:

```
 $ taskman stop a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001
 Task a7da14c7-b47a-4535-a263-5bb26e503002 stop request received.

```

#### stream
The stream command will provide users with a stream of the output given the provided `task-id`. If the task is running, the user will receive the output as it is produced continuously starting from the top of the output. If the task is not running, the user will receive the output until the end of the output stream and then close. If the task is running and the task completes or is stopped; the stream will receive the final output and be closed. If the task does not exist or is not associated with the user a `NOT_FOUND` will be returned.

```
Usage:
  taskman stream <task-id> --user-id <user-id> [--help]

Description:
  Stream real-time output from a running task identified by its unique task ID.
  This command continuously sends the task's stdout and stderr output to your terminal.

Arguments:
  <task-id>
        The unique identifier (UUID) of the task from which to stream output.
        Example: a7da14c7-b47a-4535-a263-5bb26e503002

Options:
  --user-id <user-id>
        The user or client ID issuing the request (e.g., client001). This flag is required.
  --help
        Display help information for the stream command.

Example:
  $ taskman stream a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001


```

Example

```
$ taskman stream a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001
file1.txt          file2.txt               file3.txt               file4.txt    file56.txt
```

### Library
A reusable library will be implemented with functionality to start, stop, get-status, and stream jobs. A gRPC server will wrap this functionality of the library and expose the APIs.

#### Process Execution Lifecycle
Tasks will be created using the go package `os/exec` to create child processes. The child process will run asynchronously until it completes or is stopped by the user calling the stop CLI command. Each process will be added to its own cgroup such as `/sys/fs/cgroup/<group_name>/cgroup.procs` and the process will be limited to the quotas set for the cgroup.

By leveraging the `syscall` library we can set the [SysProcAttr](https://pkg.go.dev/syscall#SysProcAttr) to have `UseCgroupFD` and `CgroupFD` set. This will put the child process into the provided cgroup file descriptor during the fork phase before the exec call; avoiding any time where the child process executes outside of the cgroup. This requires the use of cgroup v2, go 1.22 or higher, and linux kernel 5.7 or higher.

When the stop CLI command is executed, the library will first try and cleanly terminate the process with a SIGTERM but if the process is still running after a grace period a SIGKILL will be sent.

#### Cgroups
The library will utilize cgroups v2 and use the cpu, memory, and io cgroup controllers. The library code will create a cgroup for each task named after the task's uuid (e.g., /sys/fs/cgroup/a7da14c7-b47a-4535-a263-5bb26e503002). The cpu and io cgroup controllers will be configured with absolute bandwidth control and the memory controller will be configured with a max limit.

* CPU Bandwidth example configuration
	* Quota:  200000 (in microseconds)
	* Period: 1000000 (in microseconds)
	* These values will be written to `/sys/fs/cgroup/<group_name>/cpu.max`
* Memory Max example configuration
	* Memory.Max: 512M (megabytes)
	* This value will be written to `/sys/fs/cgroup/<group_name>/memory.max`
* IO Bandwidth example configuration
	* Device Identifier: 8:0
	* Max Read Bandwidth (rbps): 268435456 (represents 256 MB/s)
	* Max Write Bandwidth (wbps): 268435456 (represents 256 MB/s)
	* These values will be written to `/sys/fs/cgroup/<group_name>/io.max`

The actual values may be different in the actual implementation of the library. These values will be hard coded and not configurable for the exercise. The IO bandwidth restriction will be limited to just one block device with device number Major:Minor 8:0.

#### Streaming
Streaming will be implemented to support multiple clients streaming the output of a task at the same time and each stream call will start at the top of the output of the task. The output stream will combine the stdout and stderr process outputs and capture their output in goroutines. In a production system we would consider allowing the user to specify if they want stdout or stderr streams separately.

When a new stream request is made, the user will "subscribe" to the task's output stream and receive updates through a Go channel. These updates will be pushed to each user through their channel so they receive real-time updates. The stream will remain open until the process terminates. A rate limiter will be used to throttle the frequency of messages sent, ensuring that bursts of output do not overwhelm the client or network. Additionally, mutexes will be used to protect shared data to avoid data races and potential deadlocks.

If the task has already completed and the user calls stream task they will receive the output in full and then the stream will terminate at EOF.

#### Availability, Resilience, Scalability
The implementation of the library will not be focused on being highly available or a fault tolerant, resilient system. Tasks will be stored in memory and if running will be lost or stranded if the gRPC server crashes or is shut down. In a production system, we would want to store the tasks in a data store and reconcile on gRPC server start up. As well as follow best practices for data store backup and replication.

The exercise will only have one server instance. If this were a real system, the gRPC server should have multiple replicas gated behind load balancers and could deploy tasks to a pool of servers. These tasks could be created and added to a message queue (like Kafka) and picked up by the pool of servers. This would allow decoupling of the servers running the gRPC server and allow the processes to run on different servers that could have less access and privileges and be more isolated.

The library will also not be sanitizing input or ensuring that commands being run are "safe" to run on the system. The commands being sent and ran could be malicious or dangerous to the server running them. In a real system, we would want to sanitize the input as much as possible and run the processes in a sandboxed environment that has limited resources and access such as a container or a chroot jail.


### Security

#### Authentication
The gRPC server and client will utilize mutual TLS (mTLS) authentication where the client and server will authenticate one another to set up a trusted connection. The server and clients will each get their own certificates to use. 

The TLS configuration for the server will be set up with a modern and recommended configuration. The server will enforce TLS version 1.3 as it boasts improvements over 1.2 and we do not have any legacy clients for this service that may only support version 1.2. Below is the TLS config that will be used for the gRPC server:

```
tlsConfig := &tls.Config{
	Certificates: []tls.Certificate{serverCert},
	ClientAuth:   tls.RequireAndVerifyClientCert,
	ClientCAs:    caCertPool,
	MinVersion:   tls.VersionTLS13,
	CurvePreferences: []tls.CurveID{
        tls.X25519,
        tls.CurveP256,
        tls.CurveP384,
    },
}
```
Since only TLS version 1.3 is being supported we do not need to include a list of CipherSuites because they are fixed in Go see [crypto/tls](https://github.com/golang/go/blob/master/src/crypto/tls/common.go#L688-L697).

The client and server certificates will be signed by a CA that is generated locally. This CA certificate will be self signed and not by a trusted third party. These certificates will be committed to the exercise repository as a proof of concept. In a production environment, we would want these certificates to be generated, short-lived, and renewed through automation securely.

#### Authorization
Each certificate will be generated with a Common Name (CN) that will identify the user. The certificate's common name will be extracted by the gRPC server and associated with each created task. When a user makes a request against a task their common name will be extracted from the provided certificate on the request and must match the name associated with the task on the server in order for requests to be accepted.

There will be an admin certificate that the gRPC server allows access to all tasks and operations on the server.

### Privacy
The gRPC server will only store data in memory for the life of the server and will be lost on server shutdown or restart. Any other privacy considerations are beyond the scope of this exercise.

### Proto Specification
This exercise will have a new protobuf representing the TaskManager service that exposes functionality to start, stop, get, and stream tasks.

```package task_manager;

// TaskManager is a service that exposes APIs to manage linux processes (tasks)
service TaskManager {
    // StartTask takes a task and arguments and starts a new task
    rpc StartTask (StartTaskRequest) returns (StartTaskResponse);
    // StopTask stops a running task by task ID
    rpc StopTask (StopTaskRequest) returns (StopTaskResponse);
    // GetTaskStatus gets the status of a task by task ID
    rpc GetTaskStatus (TaskStatusRequest) returns (TaskStatusResponse);
    // StreamTaskOutput streams the output of a task by task ID
    rpc StreamTaskOutput (StreamTaskOutputRequest) returns (stream StreamTaskOutputResponse);
}

// StartTaskRequest contains the command and arguments to start a new task
message StartTaskRequest {
    // must be full path to the task to run e.g. "/bin/ls"
    string task_path = 1;
    // arguments to pass to the task e.g. ["-l", "-a"]
    repeated string args = 2;
}

message StartTaskResponse {
    // UUID v4 ID of the task generated by the server
    string task_id = 1;
}

message StopTaskRequest {
    // UUID v4 ID of the task generated by the server
    string task_id = 1;
}

// StopTaskResponse returns true if the stop task request was accepted; false otherwise
message StopTaskResponse {
    bool success = 1;
}

message TaskStatusRequest {
    // UUID v4 ID of the task generated by the server
    string task_id = 1;
}

message TaskStatusResponse {
    // UUID v4 ID of the task generated by the server
    string task_id = 1;
    // Exit code of the task; only set if task is not running
    int32 exit_code = 2;
    // PID of the process picked by the server
    string process_id = 3;
    // true if the task is currently running; false otherwise 
    bool running = 4;
    // Timestamp when the task started
    google.protobuf.Timestamp start_time = 5;
    // Timestamp when the task ended; only set if task is not running
    google.protobuf.Timestamp end_time = 6;
}

message StreamTaskOutputRequest {
    // UUID v4 ID of the task generated by the server
    string task_id = 1;
}

// StreamTaskOutputResponse contains stdout and stderr output from the task
// Each output line is sent as a separate message
message StreamTaskOutputResponse {
    string output = 1;
}
```

### Audit Events
Audit events would be important to consider and include for a production server implementing arbitrary linux processes they will not be included in this exercise.

### Observability
Observability features such as metrics, tracing, and logging would be included in a full-fledged production system; they will be considered beyond the scope of this exercise. Some logging may be printed to the console.

### Product Usage
Product usage events and telemetry will also be considered beyond the scope of this exercise but would be important to include in a production ready system.
