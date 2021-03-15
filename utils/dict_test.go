package utils

import (
	"reflect"
	"testing"
)

func Test_shift(t *testing.T) {
	path := "a.b.c"
	prefix, remain := shift(path)
	if prefix != "a" || remain != "b.c" {
		t.Fatal("expected not equal")
	}
}

func Test_Delete(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 123,
			},
		},
	}
	expected := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{},
		},
	}
	Delete(data, "a.b.c")

	if !reflect.DeepEqual(data, expected) {
		t.Fatal("expected not equal")
	}
}

func Test_Set(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 123,
			},
		},
	}
	expected := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 456,
			},
		},
	}

	Set(data, "a.b.c", 456)

	if !reflect.DeepEqual(data, expected) {
		t.Fatal("expected not equal")
	}
}

func Test_Get(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 123,
			},
		},
	}
	value := Get(data, "a.b.c")
	if value.(int) != 123 {
		t.Fatal("expected not equal")
	}
}
