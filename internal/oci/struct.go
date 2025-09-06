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

import v1 "github.com/opencontainers/image-spec/specs-go/v1"

type OciImage struct {
	Path               string
	LayoutVersion      string
	PextraImageType    string
	Index              *v1.Index
	SelectedDescriptor *v1.Descriptor
	Manifest           *v1.Manifest
	Config             *v1.Image
}

type manifestDesc struct {
	v1.Descriptor
	imageType string
}
