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
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

func requireTar(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("system tar not found; skipping")
	}
}

type tarEntry struct {
	Name    string
	Mode    int64
	Type    byte
	Content []byte
}

func writeUncompressedTar(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	now := time.Now()
	for _, e := range entries {
		h := &tar.Header{
			Name:    e.Name,
			Mode:    e.Mode,
			Size:    int64(len(e.Content)),
			ModTime: now,
		}
		if e.Type == 0 {
			h.Typeflag = tar.TypeReg
			if h.Mode == 0 {
				h.Mode = 0644
			}
		} else {
			h.Typeflag = e.Type
			if h.Typeflag == tar.TypeDir {
				if !strings.HasSuffix(h.Name, "/") {
					h.Name += "/"
				}
				h.Size = 0
			}
			if h.Mode == 0 {
				h.Mode = 0755
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

func TestPlanWhiteouts_Uncompressed(t *testing.T) {
	requireTar(t)

	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "lxc.tar")

	entries := []tarEntry{
		{Name: "regular.txt"},
		{Name: "dira/", Type: tar.TypeDir},
		{Name: filepath.Join("dira", OpaqueDirMarker)},
		{Name: filepath.Join("dirb", ".wh.fileA")},
		{Name: filepath.Join("dirb", "sub", ".wh.fileB")},
	}
	writeUncompressedTar(t, tarPath, entries)

	opq, wh, err := planWhiteouts(tarPath, pextraoci.MediaTypePextraImageLayerLxc)
	if err != nil {
		t.Fatalf("planWhiteouts error: %v", err)
	}

	if _, ok := opq["dira"]; !ok {
		t.Fatalf("expected opaque dir 'dira' to be present: %v", opq)
	}
	wantWh := []string{
		filepath.Join("dirb", "fileA"),
		filepath.Join("dirb", "sub", "fileB"),
	}
	// order-insensitive compare
	gotWh := append([]string(nil), wh...)
	if !sameStringSet(gotWh, wantWh) {
		t.Fatalf("whiteouts mismatch\ngot:  %v\nwant: %v", gotWh, wantWh)
	}
}

func TestPlanSanitizedExcludes_Uncompressed(t *testing.T) {
	requireTar(t)

	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "bad.tar")

	entries := []tarEntry{
		{Name: "/abs/file"},
		{Name: "../../etc/passwd"},
		{Name: "dir/../evil"},
		{Name: "safe/file"},
	}
	writeUncompressedTar(t, tarPath, entries)

	ex, err := planSanitizedExcludes(tarPath, pextraoci.MediaTypePextraImageLayerLxc)
	if err != nil {
		t.Fatalf("planSanitizedExcludes error: %v", err)
	}

	// raw lines are expected
	expect := map[string]bool{
		"/abs/file":        true,
		"../../etc/passwd": true,
		"dir/../evil":      true,
		"safe/file":        false,
	}
	for k, want := range expect {
		found := slices.Contains(ex, k)
		if want && !found {
			t.Errorf("expected exclude to include %q; got %v", k, ex)
		}
		if !want && found {
			t.Errorf("did not expect exclude to include %q; got %v", k, ex)
		}
	}
}

func TestApplyWhiteouts(t *testing.T) {
	root := t.TempDir()
	// create paths
	mkfile(t, filepath.Join(root, "a", "b.txt"))
	mkfile(t, filepath.Join(root, "c", "d", "e"))

	paths := []string{
		filepath.Join("a", "b.txt"),
		"c",
	}

	if err := applyWhiteouts(root, paths); err != nil {
		t.Fatalf("applyWhiteouts error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "a", "b.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected a/b.txt removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "c")); !os.IsNotExist(err) {
		t.Fatalf("expected c removed, got err=%v", err)
	}
}

func TestApplyOpaqueDirs(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "x"))
	mkfile(t, filepath.Join(root, "x", "foo"))
	mkfile(t, filepath.Join(root, "x", "bar"))

	// y does not exist, should be skipped
	opq := map[string]struct{}{
		"x":     {},
		"y/sub": {},
	}

	if err := applyOpaqueDirs(root, opq); err != nil {
		t.Fatalf("applyOpaqueDirs error: %v", err)
	}

	ents, err := os.ReadDir(filepath.Join(root, "x"))
	if err != nil {
		t.Fatalf("readdir x: %v", err)
	}
	if len(ents) != 0 {
		t.Fatalf("expected x to be empty after opq, got %d entries", len(ents))
	}
}

func TestBuildTarArgs_Uncompressed(t *testing.T) {
	requireTar(t)

	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	layer := filepath.Join(tmp, "layer.tar")
	writeUncompressedTar(t, layer, []tarEntry{
		{Name: "/abs/file"},
		{Name: "dir/../evil"},
		{Name: "ok"},
	})

	args, err := buildTarArgs(layer, outDir, pextraoci.MediaTypePextraImageLayerLxc)
	if err != nil {
		t.Fatalf("buildTarArgs error: %v", err)
	}

	assertHas := func(flag string) {
		if !slices.Contains(args, flag) {
			t.Fatalf("expected args to include %q; got %v", flag, args)
		}
	}
	assertHas("-C")
	assertHas(outDir)
	assertHas("-x")
	assertHas("--numeric-owner")
	assertHas("--same-permissions")
	assertHas("--delay-directory-restore")
	assertHas("--keep-directory-symlink")
	assertHas("--overwrite")
	assertHas("--xattrs")
	assertHas("--xattrs-include=*")
	assertHas("--acls")
	assertHas("--selinux")
	assertHas("--exclude=" + WhiteoutPrefix + "*")
	assertHas("--exclude=*/" + WhiteoutPrefix + "*")
	assertHas("--exclude=" + OpaqueDirMarker)
	assertHas("--exclude=*/" + OpaqueDirMarker)
	assertHas("--exclude=/abs/file")
	assertHas("--exclude=dir/../evil")

	if os.Geteuid() != 0 {
		assertHas("--no-same-owner")
	} else if slices.Contains(args, "--no-same-owner") {
		t.Fatalf("did not expect --no-same-owner when running as root")
	}

	// last pair should be "-f", layer
	if len(args) < 2 || args[len(args)-2] != "-f" || args[len(args)-1] != layer {
		t.Fatalf("expected final args to be ['-f', %q], got %v", layer, args[len(args)-2:])
	}
}

// helpers

func mkfile(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("writefile: %v", err)
	}
}

func mkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ma := map[string]int{}
	mb := map[string]int{}
	for _, x := range a {
		ma[x]++
	}
	for _, x := range b {
		mb[x]++
	}
	return reflect.DeepEqual(ma, mb)
}
