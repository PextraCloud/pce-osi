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
	"path"

	"github.com/PextraCloud/pce-osi/internal/utils"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

func (c *QemuConfig) tempDirWithOriginalFiles() (string, error) {
	tempDir, err := os.MkdirTemp("", "pce-oci-qemu-flatten-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	for _, layer := range c.Layers {
		digest := layer.Digest.String()
		originalFileName := layer.Annotations[pextraoci.AnnotationPextraQemuFileName] // TODO: validate that this exists earlier

		srcPath := utils.BlobPath(c.ImgPath, digest)
		destPath := path.Join(tempDir, originalFileName)

		// Create a symlink to the original file
		if err := os.Symlink(srcPath, destPath); err != nil {
			return "", fmt.Errorf("failed to create symlink for layer %s: %w", digest, err)
		}
	}

	return tempDir, nil
}
