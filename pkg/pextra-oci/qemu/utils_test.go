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
package qemu

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestTempDirWithOriginalFiles_SymlinksCreated(t *testing.T) {
	img := t.TempDir()
	out := t.TempDir()

	// Prepare two fake blob files
	layers := []v1.Descriptor{
		{
			MediaType:   pextraoci.MediaTypePextraImageLayerQcow2,
			Digest:      v1.Descriptor{Digest: "sha256:deadbeef"}.Digest, // quick way to get v1.Digest from string
			Annotations: map[string]string{pextraoci.AnnotationPextraQemuFileName: "disk1.qcow2"},
		},
		{
			MediaType:   pextraoci.MediaTypePextraImageLayerQcow2,
			Digest:      v1.Descriptor{Digest: "sha256:feedface"}.Digest,
			Annotations: map[string]string{pextraoci.AnnotationPextraQemuFileName: "disk2.qcow2"},
		},
	}

	// Create blob files at the locations utils.BlobPath will resolve
	for _, l := range layers {
		p := utils.BlobPath(img, l.Digest.String())
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir blobs dir: %v", err)
		}
		if err := os.WriteFile(p, []byte("qcow2"), 0o644); err != nil {
			t.Fatalf("write blob: %v", err)
		}
	}

	cfg := &QemuConfig{Layers: layers, ImgPath: img, OutputDir: out}
	tmp, err := cfg.tempDirWithOriginalFiles()
	if err != nil {
		t.Fatalf("tempDirWithOriginalFiles error: %v", err)
	}
	defer os.RemoveAll(tmp)

	// Verify symlinks point to the correct blob paths
	for _, l := range layers {
		name := l.Annotations[pextraoci.AnnotationPextraQemuFileName]
		link := filepath.Join(tmp, name)
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("lstat symlink %s: %v", link, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected %s to be a symlink", link)
		}
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("readlink %s: %v", link, err)
		}
		want := utils.BlobPath(img, l.Digest.String())
		if target != want {
			t.Fatalf("symlink target mismatch for %s: got %q want %q", link, target, want)
		}
	}
}
