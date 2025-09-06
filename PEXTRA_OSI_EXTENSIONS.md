# Pextra OSI Extensions

## Overview

-   Pextra images are standard OCI Image Layouts with additional annotations and media types.
-   A manifest must declare the Pextra image type to guide tooling (e.g., extraction).
-   Supported image types:
    -   LXC (Linux Containers)
    -   QEMU (virtual machine disk images)

## Manifest Annotation

-   Key: `org.pextra.image.type`
-   Allowed values: `lxc`, `qemu`
-   Selection:
    -   Tools select a manifest for the current platform (`GOOS`/`GOARCH`) when available; otherwise the first matching manifest is used.
    -   Nested indices are permitted; the same selection rules apply recursively.

## LXC Image

-   Layer media types:
    -   `application/vnd.pextra.image.layer.v1.lxc.tar`
    -   `application/vnd.pextra.image.layer.v1.lxc.tar+gzip`
    -   `application/vnd.pextra.image.layer.v1.lxc.tar+zstd`
-   Layer semantics (similar to Docker image layer handling):
    -   Layers are applied in manifest order.
    -   Whiteouts:
        -   Files prefixed with `.wh.` denote removal of the target path in the extracted rootfs.
        -   `.wh..wh..opq` inside a directory marks that directory as opaque (pre-existing contents removed before applying current layer).
    -   The whiteout markers themselves are not extracted into the target rootfs.
-   Security and sanitation:
    -   Archive entries that are absolute (`/...`) or contain `..` components are excluded during extraction.
    -   Extraction is performed with `--numeric-owner`, `--same-permissions`, `--delay-directory-restore`, `--keep-directory-symlink`, `--overwrite`, `--xattrs --xattrs-include=*`, `--acls`, `--selinux` (subject to `tar` support).
-   Tooling:
    -   Extraction uses the system `tar` and supports `gzip`/`zstd` according to the declared media type.

## QEMU Image

-   Layer media type: `application/vnd.pextra.image.layer.v1.qcow2`
-   Layer annotations:
    -   `org.pextra.qcow2.fileName`: Desired output file name (e.g., `disk0.qcow2`). Required.
        -   When using backing files, ensure that the parent file name is one-level (no slashes) and that the backing file, with the original name, is also included in the manifest. Otherwise, the image extraction may fail.
    -   `org.pextra.qcow2.flatten`: Optional boolean (`true`/`false`). When `true`, tooling produces a standalone qcow2 via `qemu-img convert`.
-   Behavior:
    -   Tools locate blobs by digest under the OCI `blobs/` tree.
    -   When `flatten=true`, the resulting qcow2 is written to the output directory using `org.pextra.qcow2.fileName`.
    -   When `flatten=false`, tooling may leave the blob as-is (implementation-defined whether it is copied or skipped).

## Examples (manifest snippets)

```json
{
	"mediaType": "application/vnd.oci.image.manifest.v1+json",
	"annotations": {
		"org.pextra.image.type": "lxc"
	},
	"config": { "...": "..." },
	"layers": [
		{
			"mediaType": "application/vnd.pextra.image.layer.v1.lxc.tar+zstd",
			"digest": "sha256:...",
			"size": 123
		}
	]
}
```

```json
{
	"mediaType": "application/vnd.oci.image.manifest.v1+json",
	"annotations": {
		"org.pextra.image.type": "qemu"
	},
	"layers": [
		{
			"mediaType": "application/vnd.pextra.image.layer.v1.qcow2",
			"digest": "sha256:...",
			"size": 1048576,
			"annotations": {
				"org.pextra.qcow2.fileName": "disk0.qcow2",
				"org.pextra.qcow2.flatten": "true"
			}
		}
	]
}
```

## Notes

-   All content remains valid OCI; registries and runtimes can store/transport without understanding Pextra-specific fields.
