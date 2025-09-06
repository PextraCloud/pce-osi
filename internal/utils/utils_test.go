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
package utils

import (
	"path/filepath"
	"reflect"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestSplitDigest(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		algo, hex := SplitDigest("sha256:deadbeef")
		if algo != "sha256" || hex != "deadbeef" {
			t.Fatalf("got (%q,%q), want (%q,%q)", algo, hex, "sha256", "deadbeef")
		}
	})
	t.Run("invalid_no_colon", func(t *testing.T) {
		algo, hex := SplitDigest("deadbeef")
		if algo != "" || hex != "deadbeef" {
			t.Fatalf("got (%q,%q), want (%q,%q)", algo, hex, "", "deadbeef")
		}
	})
	t.Run("empty", func(t *testing.T) {
		algo, hex := SplitDigest("")
		if algo != "" || hex != "" {
			t.Fatalf("got (%q,%q), want (%q,%q)", algo, hex, "", "")
		}
	})
}

func TestBlobPath(t *testing.T) {
	base := t.TempDir()
	got := BlobPath(base, "sha256:abc123")
	want := filepath.Join(base, v1.ImageBlobsDir, "sha256", "abc123")
	if got != want {
		t.Fatalf("BlobPath mismatch: got %q want %q", got, want)
	}
}

func TestGetLayersByMediaType(t *testing.T) {
	layers := []v1.Descriptor{
		{MediaType: "type/a"},
		{MediaType: "type/b"},
		{MediaType: "type/a"},
	}

	t.Run("single_type", func(t *testing.T) {
		got := GetLayersByMediaType(layers, "type/a")
		want := []v1.Descriptor{{MediaType: "type/a"}, {MediaType: "type/a"}}
		if !reflect.DeepEqual(types(got), types(want)) {
			t.Fatalf("got %v want %v", types(got), types(want))
		}
	})

	t.Run("multiple_types", func(t *testing.T) {
		got := GetLayersByMediaType(layers, "type/b", "type/a")
		want := []v1.Descriptor{{MediaType: "type/a"}, {MediaType: "type/b"}, {MediaType: "type/a"}}
		if !reflect.DeepEqual(types(got), types(want)) {
			t.Fatalf("got %v want %v", types(got), types(want))
		}
	})

	t.Run("none_matching", func(t *testing.T) {
		got := GetLayersByMediaType(layers, "type/c")
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})

	t.Run("no_filters", func(t *testing.T) {
		got := GetLayersByMediaType(layers /* no media types */)
		if len(got) != 0 {
			t.Fatalf("expected empty when no media types provided, got %v", got)
		}
	})
}

// helper to compare only MediaType fields
func types(d []v1.Descriptor) []string {
	out := make([]string, len(d))
	for i := range d {
		out[i] = d[i].MediaType
	}
	return out
}
