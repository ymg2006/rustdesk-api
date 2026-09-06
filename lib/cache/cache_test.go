package cache

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSimpleCache(t *testing.T) {

	type st struct {
		A string
		B string
	}

	items := map[string]interface{}{}
	items["a"] = "b"
	items["b"] = "c"

	ab := &st{
		A: "a",
		B: "b",
	}
	items["ab"] = *ab

	a := items["a"]
	fmt.Println(a)

	b := items["b"]
	fmt.Println(b)

	ab.A = "aa"
	ab2 := st{}
	ab2 = (items["ab"]).(st)
	fmt.Println(ab2, reflect.TypeOf(ab2))

}

func TestFileCacheSet(t *testing.T) {
	fc := New("file")
	err := fc.Set("123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("Write failed")
	}
}

func TestFileCacheGet(t *testing.T) {
	fc := New("file")
	err := fc.Set("123", "45156", 300)
	if err != nil {
		t.Fatalf("Write failed")
	}
	res := ""
	err = fc.Get("123", &res)
	if err != nil {
		t.Fatalf("Read failed")
	}
	fmt.Println("res", res)
}
