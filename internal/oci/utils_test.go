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
package oci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestCheckManifestAnnotations(t *testing.T) {
	t.Run("qemu", func(t *testing.T) {
		d := v1.Descriptor{Annotations: map[string]string{
			pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeQemu,
		}}
		got, ok := checkManifestAnnotations(d)
		if !ok || got != pextraoci.PextraImageTypeQemu {
			t.Fatalf("got (%v,%v), want (%v,true)", got, ok, pextraoci.PextraImageTypeQemu)
		}
	})
	t.Run("lxc", func(t *testing.T) {
		d := v1.Descriptor{Annotations: map[string]string{
			pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeLxc,
		}}
		got, ok := checkManifestAnnotations(d)
		if !ok || got != pextraoci.PextraImageTypeLxc {
			t.Fatalf("got (%v,%v), want (%v,true)", got, ok, pextraoci.PextraImageTypeLxc)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		d := v1.Descriptor{Annotations: map[string]string{
			pextraoci.AnnotationPextraImageType: "other",
		}}
		if _, ok := checkManifestAnnotations(d); ok {
			t.Fatalf("expected false for unknown type")
		}
	})
	t.Run("missing", func(t *testing.T) {
		if _, ok := checkManifestAnnotations(v1.Descriptor{}); ok {
			t.Fatalf("expected false for missing annotations")
		}
	})
}

func TestMatchesPlatform(t *testing.T) {
	goos, goarch := "linux", "amd64"

	if !matchesPlatform(nil, goos, goarch) {
		t.Fatalf("nil platform should match")
	}
	if !matchesPlatform(&v1.Platform{OS: "LINUX", Architecture: "AMD64"}, goos, goarch) {
		t.Fatalf("case-insensitive match failed")
	}
	if matchesPlatform(&v1.Platform{OS: "windows", Architecture: "amd64"}, goos, goarch) {
		t.Fatalf("unexpected match on OS mismatch")
	}
	if matchesPlatform(&v1.Platform{OS: "linux", Architecture: "arm64"}, goos, goarch) {
		t.Fatalf("unexpected match on arch mismatch")
	}
}

func TestSelectManifestDescriptor_PlatformMatch(t *testing.T) {
	idx := v1.Index{
		Manifests: []v1.Descriptor{
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    "sha256:aaa",
				Platform:  &v1.Platform{OS: "linux", Architecture: "amd64"},
				Annotations: map[string]string{
					pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeLxc,
				},
			},
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    "sha256:bbb",
				Platform:  &v1.Platform{OS: "windows", Architecture: "amd64"},
				Annotations: map[string]string{
					pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeLxc,
				},
			},
		},
	}
	md, err := selectManifestDescriptor(t.TempDir(), &idx, "linux", "amd64")
	if err != nil {
		t.Fatalf("selectManifestDescriptor error: %v", err)
	}
	if md == nil || md.Digest != "sha256:aaa" || md.imageType != pextraoci.PextraImageTypeLxc {
		t.Fatalf("unexpected selection: %+v", md)
	}
}

func TestSelectManifestDescriptor_FallbackFirst(t *testing.T) {
	idx := v1.Index{
		Manifests: []v1.Descriptor{
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    "sha256:first",
				Platform:  &v1.Platform{OS: "windows", Architecture: "arm64"},
				Annotations: map[string]string{
					pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeQemu,
				},
			},
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    "sha256:second",
				Platform:  &v1.Platform{OS: "darwin", Architecture: "arm64"},
				Annotations: map[string]string{
					pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeQemu,
				},
			},
		},
	}
	md, err := selectManifestDescriptor(t.TempDir(), &idx, "linux", "amd64")
	if err != nil {
		t.Fatalf("selectManifestDescriptor error: %v", err)
	}
	if md == nil || md.Digest != "sha256:first" {
		t.Fatalf("expected fallback to first, got %+v", md)
	}
}

func TestSelectManifestDescriptor_NestedIndex(t *testing.T) {
	base := t.TempDir()

	// Nested index blob digest
	nestedDigest := "sha256:nestedabc"

	// Compose nested index JSON with a single valid manifest
	nested := v1.Index{
		MediaType: v1.MediaTypeImageIndex,
		Manifests: []v1.Descriptor{
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    "sha256:inner",
				Platform:  &v1.Platform{OS: "linux", Architecture: "amd64"},
				Annotations: map[string]string{
					pextraoci.AnnotationPextraImageType: pextraoci.PextraImageTypeLxc,
				},
			},
		},
	}
	nb, _ := json.Marshal(nested)
	p := utils.BlobPath(base, nestedDigest)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir blobs: %v", err)
	}
	if err := os.WriteFile(p, nb, 0o644); err != nil {
		t.Fatalf("write nested blob: %v", err)
	}

	// Top-level index points to nested index
	idx := v1.Index{
		Manifests: []v1.Descriptor{
			{
				MediaType: v1.MediaTypeImageIndex,
				Digest:    digest.Digest(nestedDigest),
			},
		},
	}

	md, err := selectManifestDescriptor(base, &idx, "linux", "amd64")
	if err != nil {
		t.Fatalf("selectManifestDescriptor nested error: %v", err)
	}
	if md == nil || md.Digest != "sha256:inner" || md.imageType != pextraoci.PextraImageTypeLxc {
		t.Fatalf("unexpected nested selection: %+v", md)
	}
}

func TestSelectManifestDescriptor_NoSuitable(t *testing.T) {
	idx := v1.Index{
		Manifests: []v1.Descriptor{
			{MediaType: v1.MediaTypeImageManifest, Digest: "sha256:x"}, // no annotations
		},
	}
	if _, err := selectManifestDescriptor(t.TempDir(), &idx, "linux", "amd64"); err == nil {
		t.Fatalf("expected error when no suitable manifest found")
	}
}

func TestReadJSONFile_SuccessAndError(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "x.json")
	type X struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	want := X{A: 42, B: "ok"}
	b, _ := json.Marshal(want)
	if err := os.WriteFile(file, b, 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}
	var got X
	if err := readJSONFile(file, &got); err != nil {
		t.Fatalf("readJSONFile error: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if err := readJSONFile(filepath.Join(tmp, "missing.json"), &got); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestReadBlobJSON_SuccessAndError(t *testing.T) {
	base := t.TempDir()
	digest := "sha256:abc123"
	type Y struct {
		N string `json:"n"`
	}
	want := Y{N: "v"}
	b, _ := json.Marshal(want)
	p := utils.BlobPath(base, digest)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir blobs: %v", err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}

	var got Y
	if err := readBlobJSON(base, digest, &got); err != nil {
		t.Fatalf("readBlobJSON error: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}

	if err := readBlobJSON(base, "sha256:doesnotexist", &got); err == nil {
		t.Fatalf("expected error for missing blob")
	}
}
