package rm

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
)

func RemoveBinary(binary string) error {
	binaryPath := filepath.Join(config.KelpBin, binary)
	if utils.FileExists(binaryPath) {
		fmt.Printf("\nRemoving binary %s...", binary)
		err := os.Remove(binaryPath)
		if err != nil {
			return err
		}
	}
	return nil
}
