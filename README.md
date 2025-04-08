# taskman
prototype job worker service that provides an API to run arbitrary Linux processes

# How to build
Binaries will be placed in the bin/ directory at the project root
```
make build
```

# How to run

Running the server:

```
$ ./bin/taskman-server
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