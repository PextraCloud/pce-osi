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
package pextraoci

const (
	AnnotationPextraImageType = "org.pextra.image.type"
	PextraImageTypeQemu       = "qemu"
	PextraImageTypeLxc        = "lxc"

	// QEMU (qcow2)
	MediaTypePextraImageLayerQcow2 = "application/vnd.pextra.image.layer.v1.qcow2"
	AnnotationPextraQemuFileName   = "org.pextra.qcow2.fileName"
	AnnotationPextraQemuFlatten    = "org.pextra.qcow2.flatten"

	// LXC
	MediaTypePextraImageLayerLxc     = "application/vnd.pextra.image.layer.v1.lxc.tar"
	MediaTypePextraImageLayerLxcGzip = "application/vnd.pextra.image.layer.v1.lxc.tar+gzip"
	MediaTypePextraImageLayerLxcZstd = "application/vnd.pextra.image.layer.v1.lxc.tar+zstd"
)
