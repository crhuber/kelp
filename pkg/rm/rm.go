package rm

import (
	"crhuber/kelp/pkg/config"
	"fmt"
	"os"
	"path/filepath"
)

func RemoveBinary(binary string) error {
	fmt.Printf("\nRemoving binary %s...", binary)
	binaryPath := filepath.Join(config.KelpBin, binary)
	err := os.Remove(binaryPath)
	if err != nil {
		return err
	}
	return nil
}
