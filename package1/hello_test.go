package package1

import (
	"strings"
	"testing"
)

func TestMyHelloFunction(t *testing.T) {
	name := string("oscar")
	mess, err := MyHelloFunction(name)
	if err != nil {
		t.Error("function threw an error")
	}
	if !strings.Contains(mess, name) {
		t.Fatal("message does not contain name")
	}
}
