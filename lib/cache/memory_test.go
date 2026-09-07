package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestMemorySet(t *testing.T) {
	mc := NewMemoryCache(0)
	err := mc.Set(context.Background(), "123", "44567", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("Write failed")
	}
}

func TestMemoryGet(t *testing.T) {
	mc := NewMemoryCache(0)
	mc.Set(context.Background(), "123", "44567", 0)
	res := ""
	err := mc.Get(context.Background(), "123", &res)
	fmt.Println("res", res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	if res != "44567" {
		t.Fatalf("read error")
	}

}

func TestMemorySetExpGet(t *testing.T) {
	mc := NewMemoryCache(0)
	//mc.stopEviction()
	mc.Set(context.Background(), "1", "10", 10*time.Second)
	mc.Set(context.Background(), "2", "5", 5*time.Second)
	err := mc.Set(context.Background(), "3", "3", 3*time.Second)
	if err != nil {
		t.Fatalf("Write failed")
	}

	res := ""
	err = mc.Get(context.Background(), "3", &res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	fmt.Println("res 3", res)
	time.Sleep(4 * time.Second)
	//res = ""
	err = mc.Get(context.Background(), "3", &res)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expired read error = %v, want ErrCacheMiss", err)
	}
	fmt.Println("res 3", res)
	err = mc.Get(context.Background(), "2", &res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	fmt.Println("res 2", res)
	err = mc.Get(context.Background(), "1", &res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	fmt.Println("res 1", res)

}
func TestMemoryLru(t *testing.T) {
	mc := NewMemoryCache(18)
	mc.Set(context.Background(), "1", "1111", 10*time.Second)
	mc.Set(context.Background(), "2", "2222", 5*time.Second)
	// Read once; key 1 will be placed at the end.
	touched := ""
	if err := mc.Get(context.Background(), "1", &touched); err != nil {
		t.Fatal(err)
	}
	err := mc.Set(context.Background(), "3", "three", 3*time.Second)
	if err != nil {
		//t.Fatalf("Write failed")
	}

	res := ""
	err = mc.Get(context.Background(), "3", &res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	fmt.Println("res3", res)
	res = ""
	err = mc.Get(context.Background(), "2", &res)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("evicted read error = %v, want ErrCacheMiss", err)
	}
	fmt.Println("res2", res)
	res = ""
	err = mc.Get(context.Background(), "1", &res)
	if err != nil {
		t.Fatalf("Read failed" + err.Error())
	}
	fmt.Println("res1", res)

}
func BenchmarkMemorySet(b *testing.B) {
	mc := NewMemoryCache(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		mc.Set(context.Background(), key, value, 1000*time.Second)
	}
}
