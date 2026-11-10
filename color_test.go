package shell_test

import (
	"strings"
	"testing"

	"github.com/api-go/shell"
)

func TestStripANSI(t *testing.T) {
	input := shell.Red("hello")
	output := shell.StripANSI(input)
	if output != "hello" {
		t.Errorf("Expected 'hello', got '%s'", output)
	}
}

func TestColor(t *testing.T) {
	red := shell.Style(shell.TextRed, "hello", " world")
	if !strings.Contains(red, "hello") || !strings.Contains(red, "world") {
		t.Errorf("Style() should contain string, got %s", red)
	}
}
