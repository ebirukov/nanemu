package cpio

import (
	"bytes"
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
//	root 	- path to the root directory to archive
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
func Create(out io.Writer, root string) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return fmt.Errorf("root does not exist: %s", root)
	}
	if err != nil {
		return err
	}

	archive := cpio.NewWriter(out)
	defer archive.Close()

	if !info.IsDir() {
		// root is single file
		return addFileToArchive(archive, filepath.Dir(root), root, info)
	}

	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return addFileToArchive(archive, root, path, fi)
	})
}

func addFileToArchive(archive *cpio.Writer, dir, path string, info os.FileInfo) error {
	name, err := filepath.Rel(dir, path)
	if err != nil {
		return err
	}
	// skip root
	if name == "." {
		return nil
	}

	hdr, err := cpio.FileInfoHeader(info, name)
	if err != nil {
		return err
	}
	hdr.Name = name

	if info.Mode().IsRegular() {
		isExec, err := isExecutableFile(path)
		if err != nil {
			return err
		}
		if isExec {
			hdr.Mode = 0755
		}
	}

	if err := archive.WriteHeader(hdr); err != nil {
		return err
	}

	if info.Mode().IsDir() {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := io.Copy(archive, file); err != nil {
		return err
	}

	return nil
}

func isExecutableFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return false, err
	}

	switch {
	// ELF
	case bytes.Equal(header[:4], []byte{0x7F, 'E', 'L', 'F'}):
		return true, nil
	// script
	case bytes.Equal(header[:2], []byte{'#', '!'}):
		return true, nil
	}

	return false, nil
}
