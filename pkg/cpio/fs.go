package cpio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cavaliergopher/cpio"
)

// Create writes a CPIO archive containing the directory tree rooted at rootDir
// to the provided io.Writer.
//
// The function recursively walks the directory tree starting at rootDir and
// creates a CPIO archive with the following properties:
// - Preserves file modes and permissions
// - Handles regular files and symlinks
// - Maintains relative paths within the archive
// - Automatically closes the archive when complete
//
// Example usage:
//
//	// Create a CPIO archive from a directory
//	outFile, err := os.Create("initramfs.cpio")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer outFile.Close()
//
//	if err := cpio.Create(outFile, "./rootfs"); err != nil {
//	    log.Fatalf("Failed to create CPIO archive: %v", err)
//	}
//
// Parameters:
//
//	out     - io.Writer destination for the CPIO archive data
//	rootDir - path to the root directory to archive
//
// Returns:
//
//	error - nil on success, or one of:
//	        - os.ErrNotExist if rootDir doesn't exist
//	        - cpio specific errors for archive creation failures
//	        - filepath.Walk errors for directory traversal issues
//
// Notes:
// - The archive is always closed, even if an error occurs
// - Symlinks are preserved in the archive
// - Special files (devices, sockets etc.) are skipped
// - File ownership is preserved from the source filesystem
// - Uses standard CPIO newc format for maximum compatibility
func Create(out io.Writer, rootDir string) error {
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return fmt.Errorf("root does not exist: %s", rootDir)
	}

	archive := cpio.NewWriter(out)
	defer archive.Close()

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}

		hdr, err := cpio.FileInfoHeader(info, name)
		if err != nil {
			return err
		}
		hdr.Name = name

		if err := archive.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(archive, file); err != nil {
				return err
			}
		}

		return nil
	})
}
