package types

import (
	"runtime"
)

type OS int

const (
	Darwin OS = iota
	Linux
)

func IsDarwin() bool {
	return runtime.GOOS == "darwin"
}

func IsLinux() bool {
	return runtime.GOOS == "linux"
}

type Capabilities struct {
	OS             OS
	ExecutableMime string
	Arch           string
}

func GetOS() OS {
	if runtime.GOOS == "darwin" {
		return Darwin
	}
	if runtime.GOOS == "linux" {
		return Linux
	}
	return -1
}

var current *Capabilities

func GetCapabilities() *Capabilities {
	if current != nil {
		return current
	}

	switch runtime.GOOS {
	case "darwin":
		current = &Capabilities{OS: Darwin, ExecutableMime: "application/x-mach-binary"}
	case "linux":
		current = &Capabilities{OS: Linux, ExecutableMime: "application/x-executable"}
	}
	current.Arch = runtime.GOARCH
	return current
}
