package provider

import (
	"reflect"
	"testing"
)

func TestDiffStrings(t *testing.T) {
	add, remove := diffStrings([]string{"a", "b"}, []string{"b", "c"})
	if !reflect.DeepEqual(add, []string{"c"}) || !reflect.DeepEqual(remove, []string{"a"}) {
		t.Fatalf("got add=%v remove=%v", add, remove)
	}
	add, remove = diffStrings(nil, nil)
	if len(add) != 0 || len(remove) != 0 {
		t.Fatalf("expected empty diff, got add=%v remove=%v", add, remove)
	}
}
