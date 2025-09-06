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
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Reads and parses an OCI image from the specified path
func GetImageDetails(imagePath string) (*OciImage, error) {
	base := filepath.Clean(imagePath)
	if fi, err := os.Stat(base); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", base)
	}

	layoutFile := filepath.Join(base, v1.ImageLayoutFile)
	indexFile := filepath.Join(base, v1.ImageIndexFile)
	if _, err := os.Stat(layoutFile); err != nil {
		return nil, fmt.Errorf("missing %s file at %s: %w", v1.ImageLayoutFile, layoutFile, err)
	}
	if _, err := os.Stat(indexFile); err != nil {
		return nil, fmt.Errorf("missing %s file at %s: %w", v1.ImageIndexFile, indexFile, err)
	}

	var layout v1.ImageLayout
	if err := readJSONFile(layoutFile, &layout); err != nil {
		return nil, fmt.Errorf("parse %s: %w", v1.ImageLayoutFile, err)
	}
	if layout.Version != v1.ImageLayoutVersion {
		return nil, fmt.Errorf("unsupported layout version %q (want %q)", layout.Version, v1.ImageLayoutVersion)
	}

	// Parse image index
	var idx v1.Index
	if err := readJSONFile(indexFile, &idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", v1.ImageIndexFile, err)
	}
	if idx.MediaType != v1.MediaTypeImageIndex {
		return nil, fmt.Errorf("unsupported index mediaType %q (want %q)", idx.MediaType, v1.MediaTypeImageIndex)
	}

	if len(idx.Manifests) == 0 {
		return nil, fmt.Errorf("index contains no manifests")
	}

	// Choose manifest descriptor by platform (GOOS/GOARCH), with fallbacks.
	desc, err := selectManifestDescriptor(base, &idx, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	// Load manifest
	var manifest v1.Manifest
	if err := readBlobJSON(base, string(desc.Digest), &manifest); err != nil {
		return nil, fmt.Errorf("load manifest %s: %w", desc.Digest, err)
	}
	if manifest.MediaType != v1.MediaTypeImageManifest {
		return nil, fmt.Errorf("unsupported manifest mediaType %q", manifest.MediaType)
	}

	// Load image config
	if manifest.Config.MediaType != v1.MediaTypeImageConfig {
		return nil, fmt.Errorf("unsupported config mediaType %q", manifest.Config.MediaType)
	}
	var config v1.Image
	if err := readBlobJSON(base, string(manifest.Config.Digest), &config); err != nil {
		return nil, fmt.Errorf("load config %s: %w", manifest.Config.Digest, err)
	}

	out := &OciImage{
		Path:               base,
		LayoutVersion:      layout.Version,
		PextraImageType:    desc.imageType,
		Index:              &idx,
		SelectedDescriptor: &desc.Descriptor,
		Manifest:           &manifest,
		Config:             &config,
	}
	return out, nil
}
