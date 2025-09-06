//go:build integration

/*
Copyright 2025 Pextra Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package lxc

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

func writeGzipTar(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create gz: %v", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	now := time.Now()
	for _, e := range entries {
		h := &tar.Header{
			Name:    e.Name,
			Mode:    e.Mode,
			ModTime: now,
		}
		if e.Type == 0 {
			h.Typeflag = tar.TypeReg
			if h.Mode == 0 {
				h.Mode = 0644
			}
			h.Size = int64(len(e.Content))
		} else {
			h.Typeflag = e.Type
			if h.Mode == 0 {
				h.Mode = 0755
			}
			if h.Typeflag == tar.TypeDir {
				if !bytes.HasSuffix([]byte(e.Name), []byte("/")) {
					h.Name += "/"
				}
				h.Size = 0
			}
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatalf("write header %s: %v", e.Name, err)
		}
		if h.Typeflag == tar.TypeReg && len(e.Content) > 0 {
			if _, err := tw.Write(e.Content); err != nil {
				t.Fatalf("write content %s: %v", e.Name, err)
			}
		}
	}
}

func TestPlanWhiteouts_Gzip(t *testing.T) {
	requireTar(t)

	tmp := t.TempDir()
	p := filepath.Join(tmp, "lxc.tar.gz")
	writeGzipTar(t, p, []tarEntry{
		{Name: filepath.Join("d", OpaqueDirMarker)},
		{Name: filepath.Join("w", ".wh.a")},
		{Name: filepath.Join("w", "x", ".wh.b")},
	})

	opq, wh, err := planWhiteouts(p, pextraoci.MediaTypePextraImageLayerLxcGzip)
	if err != nil {
		t.Fatalf("planWhiteouts(gzip): %v", err)
	}

	if _, ok := opq["d"]; !ok {
		t.Fatalf("expected opaque dir 'd', got %v", opq)
	}
	if !sameStringSet(wh, []string{filepath.Join("w", "a"), filepath.Join("w", "x", "b")}) {
		t.Fatalf("unexpected whiteouts: %v", wh)
	}
}

func supportsTarZstd(t *testing.T) bool {
	t.Helper()
	cmd := exec.Command("tar", "--help")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte("--zstd"))
}

func TestPlanWhiteouts_Zstd(t *testing.T) {
	requireTar(t)
	if !supportsTarZstd(t) {
		t.Skip("system tar does not support --zstd")
	}

	tmp := t.TempDir()
	base := filepath.Join(tmp, "fs")
	if err := os.MkdirAll(filepath.Join(base, "d"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "w", "x"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// create markers
	if err := os.WriteFile(filepath.Join(base, "d", OpaqueDirMarker), nil, 0o644); err != nil {
		t.Fatalf("write opq: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "w", ".wh.a"), nil, 0o644); err != nil {
		t.Fatalf("write wh a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "w", "x", ".wh.b"), nil, 0o644); err != nil {
		t.Fatalf("write wh b: %v", err)
	}

	archive := filepath.Join(tmp, "lxc.tar.zst")
	cmd := exec.Command("tar", "--zstd", "-cf", archive, "-C", base, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("failed to create zstd tar via system tar: %v; out=%s", err, out)
	}

	opq, wh, err := planWhiteouts(archive, pextraoci.MediaTypePextraImageLayerLxcZstd)
	if err != nil {
		t.Fatalf("planWhiteouts(zstd): %v", err)
	}
	if _, ok := opq["d"]; !ok {
		t.Fatalf("expected opaque dir 'd', got %v", opq)
	}
	if !sameStringSet(wh, []string{filepath.Join("w", "a"), filepath.Join("w", "x", "b")}) {
		t.Fatalf("unexpected whiteouts: %v", wh)
	}
}

func TestPlanSanitizedExcludes_Gzip(t *testing.T) {
	requireTar(t)

	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.tar.gz")
	writeGzipTar(t, p, []tarEntry{
		{Name: "/abs/file"},
		{Name: "a/../../b"},
		{Name: "ok"},
	})
	ex, err := planSanitizedExcludes(p, pextraoci.MediaTypePextraImageLayerLxcGzip)
	if err != nil {
		t.Fatalf("planSanitizedExcludes(gzip): %v", err)
	}

	if !slices.Contains(ex, "/abs/file") || !slices.Contains(ex, "a/../../b") {
		t.Fatalf("expected excludes for unsafe paths, got %v", ex)
	}
	if slices.Contains(ex, "ok") {
		t.Fatalf("expected 'ok' to not be excluded, got %v", ex)
	}
}
func TestPlanSanitizedExcludes_Zstd(t *testing.T) {
	requireTar(t)
	if !supportsTarZstd(t) {
		t.Skip("system tar does not support --zstd")
	}

	tmp := t.TempDir()
	base := filepath.Join(tmp, "fs")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "ok"), nil, 0o644); err != nil {
		t.Fatalf("write ok: %v", err)
	}

	archive := filepath.Join(tmp, "bad.tar.zst")
	cmd := exec.Command("tar", "--zstd", "-cf", archive, "-C", base, "abs", "a", "ok")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("failed to create zstd tar via system tar: %v; out=%s", err, out)
	}

	ex, err := planSanitizedExcludes(archive, pextraoci.MediaTypePextraImageLayerLxcZstd)
	if err != nil {
		t.Fatalf("planSanitizedExcludes(zstd): %v", err)
	}
	if slices.Contains(ex, "ok") {
		t.Fatalf("expected 'ok' to not be excluded, got %v", ex)
	}
}
