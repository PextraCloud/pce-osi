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
	"fmt"
	"os"
	"strings"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Selects the most appropriate manifest descriptor from the index based on platform information
func selectManifestDescriptor(base string, idx *v1.Index, goos, goarch string) (*manifestDesc, error) {
	var manifestDescs []manifestDesc
	for _, d := range idx.Manifests {
		switch d.MediaType {
		case v1.MediaTypeImageManifest, "": // empty is tolerated by some tools
			var imageType string
			var ok bool
			if imageType, ok = checkManifestAnnotations(d); !ok {
				continue
			}
			manifestDescs = append(manifestDescs, manifestDesc{Descriptor: d, imageType: imageType})
		case v1.MediaTypeImageIndex:
			// Nested index, handle later
		default:
		}
	}

	// Platform match
	for i := range manifestDescs {
		if matchesPlatform(manifestDescs[i].Platform, goos, goarch) {
			return &manifestDescs[i], nil
		}
	}
	// Fallback to first manifest descriptor (for non-nested index)
	if len(manifestDescs) > 0 {
		return &manifestDescs[0], nil
	}

	// If no manifest descriptors found, try nested index (one level deep, last resort)
	for _, d := range idx.Manifests {
		if d.MediaType != v1.MediaTypeImageIndex {
			continue
		}

		var nested v1.Index
		if err := readBlobJSON(base, d.Digest.String(), &nested); err != nil {
			return nil, fmt.Errorf("load nested index %s: %w", d.Digest, err)
		}
		return selectManifestDescriptor(base, &nested, goos, goarch)
	}

	return nil, fmt.Errorf("no suitable manifest descriptor found")
}

// Checks for Pextra-specific annotations in the manifest descriptor
func checkManifestAnnotations(d v1.Descriptor) (string, bool) {
	if d.Annotations == nil {
		return "", false
	}
	it, ok := d.Annotations[pextraoci.AnnotationPextraImageType]
	if !ok {
		return "", false
	}
	switch it {
	case pextraoci.PextraImageTypeQemu, pextraoci.PextraImageTypeLxc:
		return it, true
	default:
		return "", false
	}
}

// Checks if the image platform matches the given GOOS and GOARCH
func matchesPlatform(p *v1.Platform, goos, goarch string) bool {
	if p == nil {
		// If platform is unspecified, many tools treat it as a wildcard.
		return true
	}
	if p.OS != "" && !strings.EqualFold(p.OS, goos) {
		return false
	}
	// Architecture comparisons can have variants; basic match first.
	if p.Architecture != "" && !strings.EqualFold(p.Architecture, goarch) {
		return false
	}
	return true
}

func readJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func readBlobJSON(base, digest string, v any) error {
	p := utils.BlobPath(base, digest)
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
