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
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

func (c *QemuConfig) FlattenQemuLayers() error {
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	layers := utils.GetLayersByMediaType(c.Layers, pextraoci.MediaTypePextraImageLayerQcow2)
	if len(layers) == 0 {
		return fmt.Errorf("no QEMU layers found in image")
	}

	// Prepare temp directory for flattening
	tempDir, err := c.tempDirWithOriginalFiles()
	if err != nil {
		return fmt.Errorf("failed to prepare temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Flatten layers that have flatten annotation
	var total int
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		flatten := layer.Annotations[pextraoci.AnnotationPextraQemuFlatten] == "true"
		if !flatten {
			continue
		}

		digest := layer.Digest.String()
		originalFileName := layer.Annotations[pextraoci.AnnotationPextraQemuFileName]
		layerPath := path.Join(tempDir, originalFileName)
		outputPath := path.Join(c.OutputDir, originalFileName)

		if err := flattenQemuLayer(layerPath, outputPath); err != nil {
			return fmt.Errorf("failed to flatten layer %s: %w", digest, err)
		}
		total++
	}

	fmt.Printf("Flattened %d/%d QEMU layers into directory %s\n", total, len(layers), c.OutputDir)
	return nil
}

func flattenQemuLayer(layerPath, outputPath string) error {
	cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", layerPath, outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
