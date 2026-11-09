package task

import (
	"reflect"
	"testing"
)

func TestGroupBy(t *testing.T) {
	got := GroupBy([]string{"apple", "bob", "avocado", "cat", "bee"}, func(s string) byte { return s[0] })
	want := map[byte][]string{'a': {"apple", "avocado"}, 'b': {"bob", "bee"}, 'c': {"cat"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	if len(GroupBy([]int{}, func(i int) int { return i })) != 0 {
		t.Fatal("empty")
	}
}
