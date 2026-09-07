package main

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/config"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/lib/cache"
)

func TestInitializeRedisCacheFailureDisablesCache(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	global.Logger = logrus.New()
	global.Redis = nil
	global.Cache = nil
	cfg := &config.Config{Cache: config.Cache{
		Type: cache.TypeRedis, RedisAddr: addr, ConnectTimeout: 50 * time.Millisecond,
	}}
	if err := initializeCache(cfg); err != nil {
		t.Fatalf("cache outage must not fail startup: %v", err)
	}
	if global.Redis != nil {
		t.Fatal("failed Redis initialization must not retain a Redis client")
	}
	if _, ok := global.Cache.(*cache.NoopCache); !ok {
		t.Fatalf("failed Redis initialization cache = %T, want *cache.NoopCache", global.Cache)
	}
}

func TestInitializeRedisCachePingsAndSelectsConfiguredDatabase(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	var mu sync.Mutex
	var commands []string
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			if len(line) < 3 || line[0] != '*' {
				return
			}
			count, parseErr := strconv.Atoi(strings.TrimSpace(line[1:]))
			if parseErr != nil {
				return
			}
			parts := make([]string, 0, count)
			for i := 0; i < count; i++ {
				lengthLine, readErr := reader.ReadString('\n')
				if readErr != nil || len(lengthLine) < 3 || lengthLine[0] != '$' {
					return
				}
				length, parseErr := strconv.Atoi(strings.TrimSpace(lengthLine[1:]))
				if parseErr != nil {
					return
				}
				data := make([]byte, length+2)
				if _, readErr := io.ReadFull(reader, data); readErr != nil {
					return
				}
				parts = append(parts, strings.ToLower(string(data[:length])))
			}
			mu.Lock()
			commands = append(commands, strings.Join(parts, " "))
			mu.Unlock()
			_, _ = writer.WriteString("+OK\r\n")
			if flushErr := writer.Flush(); flushErr != nil {
				return
			}
		}
	}()

	global.Logger = logrus.New()
	defer CloseGlobal()
	cfg := &config.Config{Cache: config.Cache{
		Type: "redis", RedisAddr: listener.Addr().String(), RedisPwd: "test-password",
		RedisDb: 1, ConnectTimeout: time.Second,
	}}
	if err := initializeCache(cfg); err != nil {
		t.Fatalf("initializeCache() error: %v", err)
	}
	if global.Redis == nil || global.Cache == nil {
		t.Fatal("Redis globals were not initialized")
	}
	if global.Redis.Options().DB != 1 {
		t.Fatalf("Redis client DB = %d, want 1", global.Redis.Options().DB)
	}

	mu.Lock()
	gotCommands := strings.Join(commands, ",")
	mu.Unlock()
	if !strings.Contains(gotCommands, "auth test-password") || !strings.Contains(gotCommands, "select 1") || !strings.Contains(gotCommands, "ping") {
		t.Fatalf("startup commands = %q, want AUTH, SELECT 1, and PING", gotCommands)
	}

	CloseGlobal()
	_ = listener.Close()
	<-serverDone
}
