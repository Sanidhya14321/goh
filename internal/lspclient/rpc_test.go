package lspclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

func TestRPCPairsOutOfOrderResponses(t *testing.T) {
	toClient, serverOut := io.Pipe()
	serverIn, toServer := io.Pipe()
	c := newRPC(toClient, toServer, nil)
	defer c.fail(io.EOF)
	defer serverIn.Close()
	defer serverOut.Close()
	serverDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(serverIn)
		requests := make([]rpcMessage, 2)
		for i := range requests {
			b, e := readFrame(reader)
			if e != nil {
				serverDone <- e
				return
			}
			if e = json.Unmarshal(b, &requests[i]); e != nil {
				serverDone <- e
				return
			}
		}
		for i := 1; i >= 0; i-- {
			r := requests[i]
			response := rpcMessage{JSONRPC: "2.0", ID: r.ID, Result: r.Params}
			b, _ := json.Marshal(response)
			if _, e := fmt.Fprintf(serverOut, "Content-Length: %d\r\n\r\n%s", len(b), b); e != nil {
				serverDone <- e
				return
			}
		}
		serverDone <- nil
	}()
	var wg sync.WaitGroup
	for _, key := range []string{"first", "second"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			result, e := c.call(ctx, "fixture/query", map[string]string{"key": key})
			if e != nil {
				t.Error(e)
				return
			}
			var value map[string]string
			if e = json.Unmarshal(result, &value); e != nil || value["key"] != key {
				t.Errorf("wrong result %s %v", result, e)
			}
		}(key)
	}
	wg.Wait()
	if e := <-serverDone; e != nil {
		t.Fatal(e)
	}
}
func TestRPCBlockedWriteReturnsOnCancellation(t *testing.T) {
	toClient, serverOut := io.Pipe()
	serverIn, toServer := io.Pipe()
	c := newRPC(toClient, toServer, nil)
	defer c.fail(io.EOF)
	defer serverIn.Close()
	defer serverOut.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, e := c.call(ctx, "blocked", map[string]string{"data": "no reader"}); e == nil {
		t.Fatal("blocked call succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("write ignored cancellation")
	}
}
