package rm

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RemoveBinary(binary string) error {
	if binary == "" {
		return nil
	}

	cleanBinDir := filepath.Clean(config.KelpBin)
	binaryPath := filepath.Join(cleanBinDir, binary)
	cleanBinaryPath := filepath.Clean(binaryPath)

	rel, err := filepath.Rel(cleanBinDir, cleanBinaryPath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return fmt.Errorf("illegal binary path %q: escapes kelp bin directory", binary)
	}

	if utils.FileExists(cleanBinaryPath) {
		logging.LogInfo("Removing binary %s...", binary)
		return os.Remove(cleanBinaryPath)
	}
	return nil
}
