# Copilot Instructions for nanemu

**nanemu** is a minimal wrapper for running ELF binaries in QEMU with a Linux kernel. It supports booting from OCI/Docker images, local directories, and custom filesystem images.

## Build, Test, and Lint

### Building
```bash
# Build the binary
go build -o nanemu ./cmd/nanemu

# Build for a specific architecture
GOOS=linux GOARCH=amd64 go build -o nanemu-amd64 ./cmd/nanemu

# Build for multiple platforms (as in CI)
GOOS=linux GOARCH=arm64 go build -o nanemu-arm64 ./cmd/nanemu
GOOS=darwin GOARCH=amd64 go build -o nanemu-darwin-amd64 ./cmd/nanemu
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/cpio
go test ./pkg/fsutil
go test ./internal/resource

# Run a single test
go test -run TestCreate ./pkg/cpio
```

Test files follow the `_test.go` naming convention. Use `t.TempDir()` for filesystem-related tests to ensure proper isolation and cleanup. The project uses `github.com/stretchr/testify` for assertions in some packages.

### Dependencies
Run `go mod tidy` after modifying dependencies. The project requires **Go 1.24.5 or later**.

## High-Level Architecture

### Core Components

**`cmd/nanemu/`** - Entry point
- Parses command-line arguments
- Initializes the App with default configuration
- Orchestrates initialization and QEMU execution

**`internal/nanemu/`** - Main application logic
- `App` struct manages QEMU execution lifecycle
- Handles kernel and rootfs resolution
- Manages QEMU process and I/O redirection
- Supports signal handling and graceful shutdown
- Executes hooks registered with `onShutdown`

**`internal/config/`** - Configuration management
- `Config` struct parses CLI flags and environment variables
- `Default` struct provides default kernel/rootfs URIs and local configuration directory
- Architecture-specific defaults are set here

**`internal/resource/`** - Resource resolution
- Fetches kernel and rootfs from various sources (OCI, Docker, local files)
- Supports remote kernel/image URIs (e.g., Alpine Linux registry)
- Handles multi-architecture support

**`internal/diskimg/`** - Disk image creation
- Creates ext4 filesystem images or cpio (initramfs) archives
- Handles temporary disk cleanup on exit
- Supports `--keep-disk` flag to preserve disk images

### Public Packages (pkg/)

**`pkg/cpio/`** - CPIO archive handling
- Creates cpio archives from directories
- Preserves file metadata (permissions, symlinks, hardlinks)
- Used for creating initramfs when ext4 is unavailable

**`pkg/ext4/`** - EXT4 filesystem utilities
- Creates and manages ext4 disk images
- Wraps `github.com/pilat/go-ext4fs`

**`pkg/fsutil/`** - Filesystem utilities
- Platform-specific inode tracking (`inode_unix.go`, `inode_windows.go`)
- Deduplicates hardlinks

**`pkg/tar/`** - TAR archive unpacking
- Unpacks TAR archives from OCI/Docker images

**`pkg/repo/`** - Repository utilities
- Handles OCI/Docker registry interactions
- Resolves image manifests and layers

## Key Conventions

### Platform-Specific Code
The project supports multiple OSes (Linux, macOS, Windows) and architectures (amd64, arm64). Platform-specific code is split into separate files:
- `_unix.go` for Unix-like systems (Linux, macOS)
- `_windows.go` for Windows
- Build tags are used to conditionally compile code (see `pkg/fsutil/inode_unix.go`)

### Modern Go Idioms
The project requires Go 1.24.5+ and follows modern Go conventions:
- Use `any` instead of `interface{}`
- Use `slices` and `maps` packages for collection operations
- Use `errors.Is()` and `errors.As()` for error handling
- Minimal use of comments—only clarify non-obvious logic

### Architecture Support
When adding support for new architectures:
- Update architecture-specific constants in `internal/nanemu/const.go`
- Test builds for all supported architectures (amd64, arm64)
- CI/CD automatically builds for linux, darwin, and windows

### Extensions System
Custom QEMU arguments can be added via the extensions mechanism without modifying source code:
- Extensions live in `$HOME/.nanemu/extension/`
- Each file name becomes a CLI flag (e.g., `my-ext` creates `-my-ext`)
- File contents are template strings that can include `%s` substitutions
- Supports architecture-specific extensions with suffixes (e.g., `my-ext-amd64`)

### Environment Variables
- `QEMU_BIN` - Path to QEMU binary (default: `qemu-system-$ARCH`)
- `QEMU_ARGS` - Additional arguments passed to QEMU
- `KERNEL_ARGS` - Additional kernel boot parameters
- `KERNEL_PATH` - Path to kernel if not provided via flag

### Testing Patterns
- Use `t.TempDir()` for temporary files in tests
- Import `github.com/stretchr/testify/assert` for assertions where used
- Test helper files are in the same package as the code being tested
- Tests verify both success paths and error handling

### Error Handling
- Wrap errors with context using `fmt.Errorf("message: %w", err)`
- Use `errors.Is()` to check for specific error types
- Log fatal errors with `log.Fatalf()` in main, return errors from functions

## CI/CD

The project uses GitHub Actions for building releases:
- Builds are triggered on git tags (e.g., `v1.0.0`)
- Builds for all combinations of `goos` (linux, windows, darwin) and `goarch` (amd64, arm64)
- Artifacts are uploaded to GitHub releases

To build a new release:
1. Create a tag: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions automatically builds and releases binaries
