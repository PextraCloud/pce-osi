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
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pce-osi",
	Short: "CLI tool for working with Pextra-specific OSI images.",
	Long: `pce-osi is a CLI for creating and managing
OCI-compliant images with Pextra-specific extensions.
It does not interact with registries.

Pextra-specific extensions to the OSI image specification
are documented at:
https://github.com/PextraCloud/pce-osi/blob/master/PEXTRA_OSI_EXTENSIONS.md.

Copyright (C) 2025 Pextra Inc. This tool is licensed
under the Apache License, Version 2.0.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
