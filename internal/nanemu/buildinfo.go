package nanemu

import (
	"debug/buildinfo"
	"fmt"
	"os"
)

func BuildInfo() (*buildinfo.BuildInfo, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("can't get executable path: %w", err)
	}

	return buildinfo.ReadFile(path)
}
