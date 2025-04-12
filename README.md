# taskman
prototype job worker service that provides an API to run arbitrary Linux processes

# How to build
Binaries will be placed in the bin/ directory at the project root
```
make build
```

# Prequisites
It is expected that your system is using Cgroup v2 and has the `io`, `memory`, and `cpu` controllers enabled for child cgroups in the cgroup.subtree_control file. These can be enabled by by running the following commands

```
echo "+io" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
echo "+memory" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
echo "+cpu" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
```

It is also assumed that you have a block device that is major:minor 8:0 and that this is /dev/sda for assigning the `io` cgroup settings. An integration test `TestIntegration_StartTaskIOThrottled` will fail if there is no /dev/sda block device.

# How to run

Running the server:

Note: Server needs sudo privileges in order to create and configure cgroups

```
$ sudo ./bin/taskman-server
```

Running CLI commands:

Start a task
```
$ ./bin/taskman --user-id client001 start -- /bin/ls /myFolder
```

Get a task status
```
$ ./bin/taskman --user-id client001 --server-address localhost:50051 get-status 123e4567-e89b-12d3-a456-426614174000
```

Stream a task output
```
$ ./bin/taskman --user-id client001 --server-address localhost:50051 stream 123e4567-e89b-12d3-a456-426614174000
```

Stop a task
```
$ ./bin/taskman --user-id client001 --server-address localhost:50053 stop 123e4567-e89b-12d3-a456-426614174000
```

# Running unit tests
Run unit tests
```
$ make unit-test
```

Run integration tests
```
$ make test-all
```

Run specific integration test
```
$ make test-integration-specific FUNC=TestIntegration_CPUThrottled_BashLoop
```

Run all tests
```
$ make test-all
```