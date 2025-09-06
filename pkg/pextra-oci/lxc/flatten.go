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
	"fmt"
	"os"
	"os/exec"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

func (c *LxcConfig) FlattenLxcLayers() error {
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filteredLayers := utils.GetLayersByMediaType(c.Layers, pextraoci.MediaTypePextraImageLayerLxc, pextraoci.MediaTypePextraImageLayerLxcGzip, pextraoci.MediaTypePextraImageLayerLxcZstd)
	if len(filteredLayers) == 0 {
		return fmt.Errorf("no LXC layers found in image")
	}

	// Extract each layer
	var total int
	for _, layer := range filteredLayers {
		digest := layer.Digest.String()
		layerPath := utils.BlobPath(c.ImgPath, digest)

		if err := c.flattenLxcLayer(layerPath, digest, layer.MediaType); err != nil {
			return fmt.Errorf("failed to flatten LXC layer %s: %w", digest, err)
		}
		total++
	}

	fmt.Printf("Extracted %d LXC layers into directory %s\n", total, c.OutputDir)
	return nil
}

func (c *LxcConfig) flattenLxcLayer(layerPath, digest, mediaType string) error {
	// Plan and apply whiteouts before extraction
	opqDirs, whiteouts, err := planWhiteouts(layerPath, mediaType)
	if err != nil {
		return fmt.Errorf("failed to list whiteouts for %s: %w", digest, err)
	}
	if err := applyOpaqueDirs(c.OutputDir, opqDirs); err != nil {
		return fmt.Errorf("failed to apply opaque dirs for %s: %w", digest, err)
	}
	if err := applyWhiteouts(c.OutputDir, whiteouts); err != nil {
		return fmt.Errorf("failed to apply whiteouts for %s: %w", digest, err)
	}

	args, err := buildTarArgs(layerPath, c.OutputDir, mediaType)
	if err != nil {
		return fmt.Errorf("failed to build tar args for %s: %w", digest, err)
	}

	cmd := exec.Command("tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract layer %s: %w", digest, err)
	}

	return nil
}
