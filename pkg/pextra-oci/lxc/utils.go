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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
)

const (
	OpaqueDirMarker = ".wh..wh..opq"
	WhiteoutPrefix  = ".wh."
)

// Scans a tar archive for whiteout entries and opaque directory markers.
func planWhiteouts(layerPath, mediaType string) (map[string]struct{}, []string, error) {
	args := []string{"-t"}
	switch mediaType {
	case pextraoci.MediaTypePextraImageLayerLxc:
		// no-op
	case pextraoci.MediaTypePextraImageLayerLxcGzip:
		args = append(args, "--gzip")
	case pextraoci.MediaTypePextraImageLayerLxcZstd:
		args = append(args, "--zstd")
	default:
		return nil, nil, fmt.Errorf("unsupported LXC layer media type: %s", mediaType)
	}
	args = append(args, "-f", layerPath)

	cmd := exec.Command("tar", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	opqDirs := make(map[string]struct{})
	var whiteouts []string

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		p := strings.TrimSpace(sc.Text())
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "./")
		p = filepath.Clean(p)

		base := filepath.Base(p)
		if base == OpaqueDirMarker {
			dir := filepath.Dir(p)
			if dir == "." {
				dir = ""
			}
			opqDirs[dir] = struct{}{}
			continue
		}
		if after, ok := strings.CutPrefix(base, WhiteoutPrefix); ok {
			target := filepath.Join(filepath.Dir(p), after)
			whiteouts = append(whiteouts, target)
		}
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Wait()
		return nil, nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, nil, err
	}
	return opqDirs, whiteouts, nil
}

// Returns archive entries that should be excluded for safety (absolute paths or with '..' components)
func planSanitizedExcludes(layerPath, mediaType string) ([]string, error) {
	args := []string{"-t"}
	switch mediaType {
	case pextraoci.MediaTypePextraImageLayerLxc:
		// no-op
	case pextraoci.MediaTypePextraImageLayerLxcGzip:
		args = append(args, "--gzip")
	case pextraoci.MediaTypePextraImageLayerLxcZstd:
		args = append(args, "--zstd")
	default:
		return nil, fmt.Errorf("unsupported LXC layer media type: %s", mediaType)
	}
	args = append(args, "-f", layerPath)

	cmd := exec.Command("tar", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var excludes []string
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		p := strings.TrimPrefix(raw, "./")

		// Absolute paths are unsafe
		if strings.HasPrefix(p, "/") {
			excludes = append(excludes, raw)
			continue
		}
		// '..' component is unsafe
		parts := strings.Split(p, "/")
		unsafe := slices.Contains(parts, "..")
		if unsafe {
			excludes = append(excludes, raw)
		}
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return excludes, nil
}

// Removes specific files/dirs listed by whiteout entries
func applyWhiteouts(root string, paths []string) error {
	for _, rel := range paths {
		target := filepath.Join(root, rel)
		if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", target, err)
		}
	}
	return nil
}

// Removes all existing entries under each opaque directory.
func applyOpaqueDirs(root string, opq map[string]struct{}) error {
	for rel := range opq {
		dir := filepath.Join(root, rel)
		ents, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading dir %s: %w", dir, err)
		}
		for _, e := range ents {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("removing %s: %w", filepath.Join(dir, e.Name()), err)
			}
		}
	}
	return nil
}

func buildTarArgs(layerPath, outputDir, mediaType string) ([]string, error) {
	unsafe, err := planSanitizedExcludes(layerPath, mediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze paths for %s: %w", layerPath, err)
	}

	args := []string{"-C", outputDir, "-x"}
	switch mediaType {
	case pextraoci.MediaTypePextraImageLayerLxc:
		// no-op
	case pextraoci.MediaTypePextraImageLayerLxcGzip:
		args = append(args, "--gzip")
	case pextraoci.MediaTypePextraImageLayerLxcZstd:
		args = append(args, "--zstd")
	default:
		return nil, fmt.Errorf("unsupported LXC layer media type: %s", mediaType)
	}

	args = append(args,
		"--numeric-owner",
		"--same-permissions",
		"--delay-directory-restore",
		"--keep-directory-symlink",
		"--overwrite",
		"--xattrs", "--xattrs-include=*",
		"--acls",
		"--selinux",
	)

	// Avoid changing ownership if not running as root
	if os.Geteuid() != 0 {
		args = append(args, "--no-same-owner")
	}

	// Exclude whiteout markers and unsafe entries from extraction
	args = append(args,
		fmt.Sprintf("--exclude=%s*", WhiteoutPrefix),
		fmt.Sprintf("--exclude=*/%s*", WhiteoutPrefix),
		fmt.Sprintf("--exclude=%s", OpaqueDirMarker),
		fmt.Sprintf("--exclude=*/%s", OpaqueDirMarker),
	)
	for _, p := range unsafe {
		args = append(args, "--exclude="+p)
	}
	args = append(args, "-f", layerPath)

	return args, nil
}
