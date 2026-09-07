package cache

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestFileSet(t *testing.T) {
	fc := NewFileCache()
	err := fc.Set(context.Background(), "123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("Write failed")
	}
}

func TestFileGet(t *testing.T) {
	fc := NewFileCache()
	res := ""
	err := fc.Get(context.Background(), "123", &res)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("Read failed")
	}
	fmt.Println("res", res)
}
func TestFileSetGet(t *testing.T) {
	fc := NewFileCache()
	err := fc.Set(context.Background(), "key1", "ddd", 0)
	res := ""
	err = fc.Get(context.Background(), "key1", &res)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("Read failed")
	}
	fmt.Println("res", res)
}
func TestFileGetJson(t *testing.T) {
	fc := NewFileCache()
	old := &r{
		A: "a", B: "b",
	}
	fc.Set(context.Background(), "123", old, 0)
	res := &r{}
	err2 := fc.Get(context.Background(), "123", res)
	fmt.Println("res", res)
	if err2 != nil {
		t.Fatalf("Read failed" + err2.Error())
	}
}
func TestFileSetGetJson(t *testing.T) {
	fc := NewFileCache()

	old_rr := &rr{AA: "aa", BB: "bb"}
	old := &r{
		A: "a", B: "b",
		R: old_rr,
	}
	err := fc.Set(context.Background(), "123", old, 300*time.Second)
	if err != nil {
		t.Fatalf("Write failed")
	}
	//old_rr.AA = "aaa"
	fmt.Println("old_rr", old)

	res := &r{}
	err2 := fc.Get(context.Background(), "123", res)
	fmt.Println("res", res)
	if err2 != nil {
		t.Fatalf("Read failed" + err2.Error())
	}
	if !reflect.DeepEqual(res, old) {
		t.Fatalf("read error")
	}

}

func BenchmarkSet(b *testing.B) {
	fc := NewFileCache()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fc.Set(context.Background(), "123", "{dsv}", 1000*time.Second)
	}
}

func BenchmarkGet(b *testing.B) {
	fc := NewFileCache()
	b.ResetTimer()
	v := ""
	for i := 0; i < b.N; i++ {
		fc.Get(context.Background(), "123", &v)
	}
}
