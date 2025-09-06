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
	"slices"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func SplitDigest(d string) (algo, hex string) {
	parts := strings.SplitN(d, ":", 2)
	if len(parts) != 2 {
		// TODO: handle this?
		return "", d
	}
	return parts[0], parts[1]
}

func BlobPath(base, digest string) string {
	algo, hex := SplitDigest(digest)
	return filepath.Join(base, v1.ImageBlobsDir, algo, hex)
}

func GetLayersByMediaType(layers []v1.Descriptor, mediaTypes ...string) []v1.Descriptor {
	var filtered []v1.Descriptor
	for _, layer := range layers {
		if slices.Contains(mediaTypes, layer.MediaType) {
			filtered = append(filtered, layer)
		}
	}
	return filtered
}
