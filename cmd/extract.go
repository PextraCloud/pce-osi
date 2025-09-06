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

	"github.com/PextraCloud/pce-osi/internal/oci"
	pextraoci "github.com/PextraCloud/pce-osi/pkg/pextra-oci"
	"github.com/PextraCloud/pce-osi/pkg/pextra-oci/lxc"
	"github.com/PextraCloud/pce-osi/pkg/pextra-oci/qemu"
	"github.com/spf13/cobra"
)

var isJson bool

func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmd.Flags().BoolVarP(&isJson, "json", "j", false, "Output information in JSON format")
}

var extractCmd = &cobra.Command{
	Use:   "extract [image-path] [output-dir]",
	Short: "Extract and flatten layers from a Pextra OSI image",
	Long: `Extracts and flattens layers from a Pextra-specific OSI image into a specified output directory.
The output directory will be created if it does not exist.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		imagePath := args[0]
		outputDir := args[1]

		res, err := oci.GetImageDetails(imagePath)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		switch res.PextraImageType {
		case pextraoci.PextraImageTypeLxc:
			c := lxc.New(res.Manifest.Layers, res.Path, outputDir)
			err = c.FlattenLxcLayers()
		case pextraoci.PextraImageTypeQemu:
			c := qemu.New(res.Manifest.Layers, res.Path, outputDir)
			err = c.FlattenQemuLayers()
		default:
			// Should never happen due to checks in oci.GetImageDetails
			err = fmt.Errorf("unsupported Pextra image type: %s", res.PextraImageType)
		}
		if err != nil {
			fmt.Println("Error extracting layers:", err)
			return
		}

		fmt.Println("Layers extracted successfully to", outputDir)
	},
}
