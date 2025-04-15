package integration

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/mikewurtz/taskman/gen/proto"
)

const streamTestTimeout = 10 * time.Second

func TestIntegration_StreamTaskOutputContextCanceled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})

	assert.Nil(t, stream)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

func TestIntegration_ShCommandOutput(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), streamTestTimeout)
	defer cancel()

	// Start a task that runs a sh command that prints three lines ex:
	//   Line 1
	//   Line 2
	//   Line 3
	startResp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "sh",
		Args:    []string{"-c", "for i in $(seq 1 3); do echo \"Line $i\"; done"},
	})

	require.NoError(t, err)
	require.NotEmpty(t, startResp.TaskId)

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: startResp.TaskId,
	})
	require.NoError(t, err)

	var allOutput []byte
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		allOutput = append(allOutput, resp.Output...)
	}

	// The expected output is exactly as printed by the sh command.
	expectedOutput := "Line 1\nLine 2\nLine 3\n"
	assert.Equal(t, expectedOutput, string(allOutput))
}

func TestIntegration_StreamTaskOutput_StdoutStderr(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), streamTestTimeout)
	defer cancel()

	startResp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "sh",
		Args:    []string{"-c", `echo "Hello, stdout"; echo "Hello, stderr" >&2`},
	})
	require.NoError(t, err)
	require.NotEmpty(t, startResp.TaskId)

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: startResp.TaskId,
	})
	require.NoError(t, err)

	var allOutput []byte
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		allOutput = append(allOutput, resp.Output...)
	}

	outputStr := string(allOutput)

	// Verify that the output includes both expected messages
	assert.Contains(t, outputStr, "Hello, stdout", "expected stdout output missing")
	assert.Contains(t, outputStr, "Hello, stderr", "expected stderr output missing")
}

func TestIntegration_ConcurrentStreamTaskOutput(t *testing.T) {
	t.Parallel()

	client1 := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), streamTestTimeout)
	defer cancel()

	startResp, err := client1.StartTask(ctx, &pb.StartTaskRequest{
		Command: "sh",
		Args:    []string{"-c", `for i in $(seq 1 5); do echo "Line $i"; sleep 0.1; done`},
	})
	require.NoError(t, err)

	stream1, err := client1.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{TaskId: startResp.TaskId})
	require.NoError(t, err)

	stream2, err := client1.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{TaskId: startResp.TaskId})
	require.NoError(t, err)

	var wg sync.WaitGroup
	readAll := func(stream pb.TaskManager_StreamTaskOutputClient, collected *string) {
		defer wg.Done()
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			*collected += string(resp.Output)
		}
	}

	var output1, output2 string
	wg.Add(2)
	go readAll(stream1, &output1)
	go readAll(stream2, &output2)
	wg.Wait()

	assert.Equal(t, output1, output2, "clients should receive identical output")
}
