package common

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestResourcesReleaseBuffersOnClose(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("maps/e1m1.txt")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte("payload")); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	zr := NewResourcesZip()
	zr.SetBuffer(buf.Bytes())
	if err := zr.Setup(); err != nil {
		t.Fatalf("ResourcesZip.Setup: %v", err)
	}
	f, err := zr.Open("maps/e1m1.txt")
	if err != nil {
		t.Fatalf("ResourcesZip.Open: %v", err)
	}
	got, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil || string(got) != "payload" {
		t.Fatalf("unexpected zip read: got=%q err=%v", string(got), err)
	}
	if err := zr.Close(); err != nil {
		t.Fatalf("ResourcesZip.Close: %v", err)
	}
	if zr.fBuffer != nil || zr.zr != nil || zr.fPath != "" {
		t.Fatalf("expected ResourcesZip fields cleared on Close, got fBuffer=%v zr=%v fPath=%q", zr.fBuffer != nil, zr.zr != nil, zr.fPath)
	}

	rm := NewResourcesMem([]byte("mem-payload"))
	if err := rm.Setup(); err != nil {
		t.Fatalf("ResourcesMem.Setup: %v", err)
	}
	if err := rm.Close(); err != nil {
		t.Fatalf("ResourcesMem.Close: %v", err)
	}
	if rm.s != nil {
		t.Fatalf("expected ResourcesMem slice released on Close")
	}

	ro := NewResourcesOs()
	if err := ro.Close(); err != nil {
		t.Fatalf("ResourcesOs.Close: %v", err)
	}

	ra := NewResourcesAsset(nil)
	if err := ra.Close(); err != nil {
		t.Fatalf("ResourcesAsset.Close: %v", err)
	}

	var _ IResources = zr
	var _ IResources = rm
	var _ IResources = ro
	var _ IResources = ra
}
