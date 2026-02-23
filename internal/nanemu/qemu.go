package nanemu

import (
	"encoding/json"
	"fmt"
	"github.com/ebirukov/nanemu/internal/diskimg"
	"github.com/ebirukov/nanemu/internal/resource"
	spec "github.com/opencontainers/image-spec/specs-go/v1"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func buildKernelBootParams(cfg *Config, rootFSBundle resource.Bundle) ([]string, error) {
	var (
		err           error
		extBootParams KernelBootParams
	)

	if rootFSBundle.MetadataPath != "" {
		extBootParams, err = extractBundleBootParams(rootFSBundle)
		if err != nil {
			fmt.Printf("Error extracting boot params from bundle metadata %s: %s\n", rootFSBundle.MetadataPath, err)
		}
	}

	bootParams := strings.Fields(getEnv(KernelArgs, defaultKernelArgs[cfg.Arch]))
	// add console UART for inspect kernel dmesg write to stdout
	if cfg.FailOnPanic && !hasFieldPrefix(bootParams, "console=") {
		bootParams = append(bootParams, defaultKernelArgs[cfg.Arch])
	}

	bootParams = append(bootParams, "panic=-1")

	if cfg.KernelBootParams.Loglevel != "" && !hasFieldPrefix(bootParams, "loglevel=") && !hasField(bootParams, "quiet") {
		bootParams = append(bootParams, "loglevel="+cfg.KernelBootParams.Loglevel)
	}

	env := mergeEnv(extBootParams.Env, cfg.KernelBootParams.Env)
	bootParams = append(bootParams, env...)
	if !hasFieldPrefix(bootParams, "PATH=") {
		bootParams = append(bootParams, "PATH=/:/bin")
	}

	if !hasFieldPrefix(bootParams, "root=") && !cfg.InitRD {
		bootParams = append(bootParams, "root=/dev/vda rw")
	}

	var initCmd string

	info, err := os.Stat(rootFSBundle.ContentPath)
	if err != nil {
		return nil, fmt.Errorf("can't get rootfs path: %w", err)
	}

	if !info.IsDir() {
		initCmd = filepath.Base(rootFSBundle.ContentPath)
	}

	if hasFieldPrefix(bootParams, "init=") || hasFieldPrefix(bootParams, "rdinit=") {
		return bootParams, nil
	}

	if len(cfg.KernelBootParams.InitCmdArgs) > 0 || len(initCmd) > 0 {
		initCmd = strings.TrimSpace(fmt.Sprintf("%s %s", initCmd, strings.Join(cfg.KernelBootParams.InitCmdArgs, " ")))
	} else {
		initCmd = strings.Join(extBootParams.InitCmdArgs, " ")
	}

	if len(initCmd) == 0 {
		return bootParams, nil
	}

	if cfg.InitRD {
		bootParams = append(bootParams, "rdinit="+initCmd)
	} else {
		bootParams = append(bootParams, "init="+initCmd)
	}

	return bootParams, nil
}

func extractBundleBootParams(bundle resource.Bundle) (KernelBootParams, error) {
	var bootParams KernelBootParams

	switch bundle.Type {
	case "web":
	case "oci", "docker":
		ociImage, err := Read[spec.Image](filepath.Join(bundle.MetadataPath, "config.json"))
		if err != nil {
			return bootParams, fmt.Errorf("can't extract oci config: %w", err)
		}
		ociCfg := ociImage.Config

		bootParams.Env = ociCfg.Env
		if len(ociCfg.Entrypoint) > 0 {
			bootParams.InitCmdArgs = ociCfg.Entrypoint
		}

		bootParams.InitCmdArgs = append(bootParams.InitCmdArgs, ociCfg.Cmd...)
	}

	return bootParams, nil
}

func buildQemuCfg(cfg *Config, rootFSFile *diskimg.ImageFile) ([]string, error) {
	qemuCfgArgs := getEnv("QEMU_ARGS", defaultQemuArgs[cfg.Arch])

	vmArgs := append(cfg.QemuExtCfgArgs.Args(), strings.Fields(qemuCfgArgs)...)

	if cfg.InitRD {
		vmArgs = append(vmArgs, "-initrd", rootFSFile.Path())
	} else {
		vmArgs = append(vmArgs, "-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", rootFSFile.Path()))
	}

	if cfg.InitRD && cfg.Memory == "" && rootFSFile.Size() > DefaultQemuMemoryBytes {
		memoryKb := int(rootFSFile.Size()+MinQemuMemoryBytes) / 1024 / 1024
		cfg.Memory = strconv.Itoa(memoryKb+1) + "M"
	}

	if cfg.Memory != "" && !hasField(vmArgs, "-m") {
		vmArgs = append(vmArgs, "-m", cfg.Memory)
	}

	if !hasField(vmArgs, "-serial") {
		vmArgs = append(vmArgs, "-serial", cfg.Serial)
	}

	if cfg.Smp != "" && !hasField(vmArgs, "-smp") {
		vmArgs = append(vmArgs, "-smp", cfg.Smp)
	}

	// accelerate hypervisor by default
	if cfg.Arch == runtime.GOARCH {
		switch runtime.GOOS {
		case "darwin":
			vmArgs = append(vmArgs, "-accel", "hvf")
		case "linux":
			vmArgs = append(vmArgs, "-enable-kvm")
		}
	}

	return vmArgs, nil
}

func hasFieldPrefix(fields []string, prefix string) bool {
	for _, f := range fields {
		if strings.Contains(f, prefix) {
			return true
		}
	}

	return false
}

func hasField(fields []string, prefix string) bool {
	for _, f := range fields {
		if strings.EqualFold(f, prefix) {
			return true
		}
	}

	return false
}

func mergeEnv(base, override []string) []string {
	env := make(map[string]string, len(base)+len(override))

	for _, e := range base {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}

	for _, e := range override {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}

func Read[T any](path string) (v *T, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("unmarshal err: %w", err)
	}

	return v, nil
}
