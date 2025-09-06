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

func TestFlattenQemuLayers_NoQemuLayers_Error(t *testing.T) {
	cfg := &QemuConfig{
		Layers:    []v1.Descriptor{{MediaType: "application/vnd.other.type"}},
		ImgPath:   t.TempDir(),
		OutputDir: t.TempDir(),
	}
	if err := cfg.FlattenQemuLayers(); err == nil {
		t.Fatalf("expected error when no QEMU layers present")
	}
}

func TestFlattenQemuLayers_NoFlatten_NoOutput(t *testing.T) {
	img := t.TempDir()
	out := t.TempDir()

	// Single qcow2 layer but not marked for flatten -> no qemu-img execution, no output produced.
	desc := v1.Descriptor{
		MediaType: pextraoci.MediaTypePextraImageLayerQcow2,
		Digest:    v1.Descriptor{Digest: "sha256:beadfeed"}.Digest,
		Annotations: map[string]string{
			pextraoci.AnnotationPextraQemuFileName: "disk.qcow2",
			pextraoci.AnnotationPextraQemuFlatten:  "false",
		},
	}

	// Create source blob file
	src := utils.BlobPath(img, desc.Digest.String())
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir blobs dir: %v", err)
	}
	if err := os.WriteFile(src, []byte("qcow2"), 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}

	cfg := &QemuConfig{Layers: []v1.Descriptor{desc}, ImgPath: img, OutputDir: out}
	if err := cfg.FlattenQemuLayers(); err != nil {
		t.Fatalf("FlattenQemuLayers error: %v", err)
	}

	// Ensure output file does not exist because we had no flattening requested.
	if _, err := os.Stat(filepath.Join(out, "disk.qcow2")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected no output file when flatten=false; stat err=%v", err)
	}
}
