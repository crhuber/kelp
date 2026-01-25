package rm

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/utils"
	"os"
	"path/filepath"
)

func RemoveBinary(binary string) error {
	binaryPath := filepath.Join(config.KelpBin, binary)
	if utils.FileExists(binaryPath) {
		logging.LogInfo("Removing binary %s...", binary)
		return os.Remove(binaryPath)
	}
	return nil
}
