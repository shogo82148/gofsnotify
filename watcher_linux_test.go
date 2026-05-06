//go:build linux

package fsnotify

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const eventTimeout = 2 * time.Second

func newWatcher(t *testing.T) *Watcher {
	t.Helper()
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func waitOp(t *testing.T, w *Watcher, want Op) Event {
	t.Helper()
	deadline := time.NewTimer(eventTimeout)
	defer deadline.Stop()
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				t.Fatalf("Events channel closed while waiting for %s", want)
			}
			if ev.Op.Has(want) {
				return ev
			}
		case err := <-w.Errors:
			t.Fatalf("unexpected error: %v", err)
		case <-deadline.C:
			t.Fatalf("timeout waiting for %s", want)
		}
	}
}

func TestWatchCreate(t *testing.T) {
	dir := t.TempDir()
	w := newWatcher(t)
	if err := w.Add(dir, Create); err != nil {
		t.Fatalf("Add: %v", err)
	}

	target := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ev := waitOp(t, w, Create)
	if ev.Name != target {
		t.Errorf("Name = %q, want %q", ev.Name, target)
	}
}

func TestWatchWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w := newWatcher(t)
	if err := w.Add(dir, Write); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := os.WriteFile(target, []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ev := waitOp(t, w, Write)
	if ev.Name != target {
		t.Errorf("Name = %q, want %q", ev.Name, target)
	}
}

func TestWatchRemove(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w := newWatcher(t)
	if err := w.Add(dir, Remove); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := os.Remove(target); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	ev := waitOp(t, w, Remove)
	if ev.Name != target {
		t.Errorf("Name = %q, want %q", ev.Name, target)
	}
}

func TestAddDuplicate(t *testing.T) {
	dir := t.TempDir()
	w := newWatcher(t)
	if err := w.Add(dir, All); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := w.Add(dir, All); !errors.Is(err, ErrAlreadyAdded) {
		t.Fatalf("Add(dup) = %v, want ErrAlreadyAdded", err)
	}
}

func TestRemoveUnregistered(t *testing.T) {
	dir := t.TempDir()
	w := newWatcher(t)
	if err := w.Remove(dir); !errors.Is(err, ErrNotAdded) {
		t.Fatalf("Remove(missing) = %v, want ErrNotAdded", err)
	}
}

func TestClosedWatcher(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close (idempotent): %v", err)
	}
	if err := w.Add(t.TempDir(), All); !errors.Is(err, ErrClosed) {
		t.Fatalf("Add after Close = %v, want ErrClosed", err)
	}
	if err := w.Remove("/tmp"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Remove after Close = %v, want ErrClosed", err)
	}

	deadline := time.NewTimer(eventTimeout)
	defer deadline.Stop()
	select {
	case _, ok := <-w.Events:
		if ok {
			t.Fatalf("Events channel should be closed")
		}
	case <-deadline.C:
		t.Fatalf("Events channel not closed after Close")
	}
}

func TestOpFilterIgnoresOthers(t *testing.T) {
	dir := t.TempDir()
	w := newWatcher(t)
	if err := w.Add(dir, Create); err != nil {
		t.Fatalf("Add: %v", err)
	}

	target := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	waitOp(t, w, Create)

	if err := os.WriteFile(target, []byte("y"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	timer := time.NewTimer(300 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case ev := <-w.Events:
			if ev.Op&^Create != 0 {
				t.Fatalf("got disallowed event %s", ev)
			}
		case <-timer.C:
			return
		}
	}
}
