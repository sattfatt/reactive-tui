package widget

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/satyam/reactive-tui/style"
)

func TestNeovimEditorInterfaces(t *testing.T) {
	ne := NewNeovimEditor("go", nil)

	// Node interface
	var _ Node = ne

	// Editable
	if !ne.IsEditable() {
		t.Error("NeovimEditor should be editable")
	}

	// EscapeThreshold
	if ne.EscapesToExit() != 3 {
		t.Errorf("EscapesToExit = %d, want 3", ne.EscapesToExit())
	}

	// Focusable
	if !ne.Focusable() {
		t.Error("NeovimEditor should be focusable")
	}
}

func TestNeovimEditorDefaults(t *testing.T) {
	ne := NewNeovimEditor("python", nil)

	if ne.filetype != "python" {
		t.Errorf("filetype = %q, want %q", ne.filetype, "python")
	}
	if ne.running {
		t.Error("should not be running initially")
	}
	if ne.term != nil {
		t.Error("term should be nil initially")
	}
	if ne.Style.Border != style.BorderSingle {
		t.Error("should have single border by default")
	}
}

func TestNeovimEditorTextRoundtrip(t *testing.T) {
	ne := NewNeovimEditor("go", nil)
	ne.SetText("hello world")
	if ne.Text() != "hello world" {
		t.Errorf("Text() = %q, want %q", ne.Text(), "hello world")
	}
}

func TestNeovimEditorTempFile(t *testing.T) {
	ne := NewNeovimEditor("go", nil)
	ne.content = "package main\n\nfunc main() {}\n"

	if err := ne.createTempFile(); err != nil {
		t.Fatalf("createTempFile: %v", err)
	}
	defer ne.cleanupTempFile()

	// Check file exists with correct extension
	if filepath.Ext(ne.tmpFile) != ".go" {
		t.Errorf("tmpFile ext = %q, want .go", filepath.Ext(ne.tmpFile))
	}

	// Read back content
	data, err := os.ReadFile(ne.tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != ne.content {
		t.Errorf("file content = %q, want %q", string(data), ne.content)
	}

	// readTempFile should match
	content, err := ne.readTempFile()
	if err != nil {
		t.Fatalf("readTempFile: %v", err)
	}
	if content != ne.content {
		t.Errorf("readTempFile = %q, want %q", content, ne.content)
	}

	// Cleanup
	dir := ne.tmpDir
	ne.cleanupTempFile()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("tmpDir should be removed after cleanup")
	}
	if ne.tmpFile != "" || ne.tmpDir != "" {
		t.Error("tmpFile and tmpDir should be empty after cleanup")
	}
}

func TestNeovimEditorTempFileDefaultExt(t *testing.T) {
	ne := NewNeovimEditor("", nil)
	ne.content = "hello"

	if err := ne.createTempFile(); err != nil {
		t.Fatalf("createTempFile: %v", err)
	}
	defer ne.cleanupTempFile()

	if filepath.Ext(ne.tmpFile) != ".txt" {
		t.Errorf("default ext = %q, want .txt", filepath.Ext(ne.tmpFile))
	}
}

func TestNeovimEditorHandleKeyWhenNotRunning(t *testing.T) {
	ne := NewNeovimEditor("go", nil)
	ev := KeyEvent{Key: int('a'), Rune: 'a'}
	if ne.HandleKey(ev) {
		t.Error("HandleKey should return false when not running")
	}
}
