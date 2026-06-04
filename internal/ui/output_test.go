package ui

import (
	"bytes"
	"os"
	"testing"

	"github.com/fatih/color"
)

func TestWithOutputCapturesColorPrintf(t *testing.T) {
	var buf bytes.Buffer
	oldOutput := Output
	oldColorOutput := color.Output

	WithOutput(&buf, func() {
		CurrentTheme.Primary.Printf("captured %s", "text")
	})

	if got := buf.String(); got == "" || !bytes.Contains([]byte(got), []byte("captured text")) {
		t.Fatalf("expected color output to be captured, got %q", got)
	}
	if Output != oldOutput {
		t.Fatalf("expected ui output to be restored")
	}
	if color.Output != oldColorOutput && color.Output != os.Stdout {
		t.Fatalf("expected color output to be restored")
	}
}
