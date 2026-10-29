package rm

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/utils"
	"fmt"
	"os"
)

func RemoveBinary(binary string) error {
	if binary == "" {
		return nil
	}

	cleanBinaryPath, err := utils.Within(config.KelpBin, binary)
	if err != nil {
		return fmt.Errorf("illegal binary path: %w", err)
	}

	if utils.FileExists(cleanBinaryPath) {
		logging.LogInfo("Removing binary %s...", binary)
		return os.Remove(cleanBinaryPath)
	}
	return nil
}
