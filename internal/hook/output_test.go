package hook

import (
	"bytes"
	"os"
	"testing"
)

func TestWriteOutputNilIsNoOp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	if err := WriteOutput(nil); err != nil {
		t.Fatalf("WriteOutput(nil) returned error: %v", err)
	}

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Errorf("WriteOutput(nil) wrote %d bytes to stdout, want 0: %q", buf.Len(), buf.String())
	}
}

func TestWriteOutputEmptyMessageIsNoOp(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	if err := WriteOutput(&Output{}); err != nil {
		t.Fatalf("WriteOutput(&Output{}) returned error: %v", err)
	}

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Errorf("WriteOutput(&Output{}) wrote %d bytes to stdout, want 0: %q", buf.Len(), buf.String())
	}
}
