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
package qemu

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func requireQemuImg(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found; skipping integration test")
	}
}

func TestFlattenQemuLayers_WithQemuImg(t *testing.T) {
	requireQemuImg(t)

	img := t.TempDir()
	out := t.TempDir()

	// Prepare a real qcow2 blob using qemu-img create
	desc := v1.Descriptor{
		MediaType: pextraoci.MediaTypePextraImageLayerQcow2,
		Digest:    v1.Descriptor{Digest: "sha256:cafebabe"}.Digest,
		Annotations: map[string]string{
			pextraoci.AnnotationPextraQemuFileName: "disk.qcow2",
			pextraoci.AnnotationPextraQemuFlatten:  "true",
		},
	}

	blob := utils.BlobPath(img, desc.Digest.String())
	if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
		t.Fatalf("mkdir blobs dir: %v", err)
	}
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", blob, "1M")
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("failed to create qcow2 blob: %v; out=%s", err, string(outb))
	}

	cfg := &QemuConfig{Layers: []v1.Descriptor{desc}, ImgPath: img, OutputDir: out}
	if err := cfg.FlattenQemuLayers(); err != nil {
		t.Fatalf("FlattenQemuLayers error: %v", err)
	}

	// Verify output file exists and is non-empty
	outFile := filepath.Join(out, "disk.qcow2")
	st, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if st.Size() == 0 {
		t.Fatalf("expected non-empty output file")
	}
}
