package cache

import (
	"bufio"
	"context"
	"github.com/go-redis/redis/v8"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRedisCacheInvalidJSONReturnsError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		line, readErr := reader.ReadString('\n')
		if readErr != nil || !strings.HasPrefix(line, "*") {
			return
		}
		count, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
		for i := 0; i < count; i++ {
			lengthLine, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			length, _ := strconv.Atoi(strings.TrimSpace(lengthLine[1:]))
			data := make([]byte, length+2)
			if _, readErr := io.ReadFull(reader, data); readErr != nil {
				return
			}
		}
		_, _ = conn.Write([]byte("$8\r\nnot-json\r\n"))
	}()

	client := redis.NewClient(&redis.Options{
		Addr: listener.Addr().String(), ReadTimeout: time.Second, WriteTimeout: time.Second,
	})
	defer client.Close()
	c := NewRedisWithClient(client)
	var value map[string]interface{}
	if err := c.Get(context.Background(), "bad", &value); err == nil {
		t.Fatal("invalid cached JSON unexpectedly decoded")
	}
	wg.Wait()
}

func TestNewRedisWithClientReusesConfiguredClient(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "dragonfly:6379",
		Password: "test-password",
		DB:       1,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisWithClient(client)
	if cache.rdb != client {
		t.Fatal("Redis cache did not retain the shared client")
	}
	if got := cache.rdb.Options().DB; got != 1 {
		t.Fatalf("Redis logical database = %d, want 1", got)
	}
}
