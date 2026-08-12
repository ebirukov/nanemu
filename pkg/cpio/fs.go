package cpio

import (
	"fmt"
	"github.com/cavaliergopher/cpio"
	"io"
	"os"
	"path/filepath"

	"github.com/ebirukov/nanemu/pkg/fsutil"
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
//   - The archive is always closed, even if an error occurs
//   - Symlinks are preserved in the archive
//   - Special files (devices, sockets etc.) are skipped
//   - File ownership is preserved from the source filesystem
//   - Uses standard CPIO newc format for maximum compatibility
//   - Executable permission bits (`execPermBits`) are applied only to regular files
//     identified as ELF binaries on scripts (based on file header), and do not override other
//     permission bits unless explicitly set
//
// Why `execPermBits` is needed:
//
// On platforms like Windows, file permission bits (such as execute) are not reliably
// preserved or even available via the filesystem. As a result, when building an
// initramfs archive on Windows, executables like Linux ELF binaries may lose their
// execute permission inside the archive, leading to runtime errors such as:
//
//	error -13 (Permission denied)
//
// This flag allows the caller to explicitly control which permission bits to apply
// to detected executable files when packaging the archive. This ensures the resulting
// initramfs is bootable and usable across platforms, regardless of the host OS
// limitations.
func Create(out io.Writer, root string, execPermBits int) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return fmt.Errorf("root does not exist: %s", root)
	}
	if err != nil {
		return err
	}

	archive := cpio.NewWriter(out)
	defer archive.Close()

	seen := make(map[fsutil.Inode]string)

	if !info.IsDir() {
		// root is single file
		return addFileToArchive(archive, filepath.Dir(root), root, info, execPermBits, seen)
	}

	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return addFileToArchive(archive, root, path, fi, execPermBits, seen)
	})
}

func addFileToArchive(archive *cpio.Writer, dir, path string, info os.FileInfo, execPermBits int, seen map[fsutil.Inode]string) error {
	name, err := filepath.Rel(dir, path)
	if err != nil {
		return fmt.Errorf("cpio: could not determine relative path for %s: %w", path, err)
	}
	// skip root
	if name == "." {
		return nil
	}

	name = filepath.ToSlash(name)
	hdr, err := cpio.FileInfoHeader(info, name)
	if err != nil {
		return fmt.Errorf("failed to create cpio header for %s: %w", name, err)
	}
	hdr.Name = name

	// Unix, Linux, macOS
	if key, ok := fsutil.InodeKeyOf(info); ok {
		if prev, seenBefore := seen[key]; seenBefore {
			hdr.Linkname = prev
			hdr.Size = 0
		} else {
			seen[key] = hdr.Name
		}
		hdr.Inode = key.Ino
		hdr.Links = key.Nlink
	}

	if info.Mode().IsRegular() && execPermBits > 0 {
		isExec, err := fsutil.IsExecutableFile(path)
		if err != nil {
			return err
		}
		if isExec {
			// add exec attributes
			fmt.Printf("file %s permision mode flags %v", path, hdr.Mode)
			hdr.Mode |= cpio.FileMode(execPermBits)
			fmt.Printf(" was modified to %v\n", hdr.Mode)
		}
	}

	if err := archive.WriteHeader(hdr); err != nil {
		return fmt.Errorf("cpio: failed to write file header for %s: %w", name, err)
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("cpio: failed to write %s %s to archive :%w", info.Name(), info.Mode(), err)
		}

		if _, err = archive.Write([]byte(linkTarget)); err != nil {
			return fmt.Errorf("cpio: failed to write %s %s to archive :%w", info.Name(), info.Mode(), err)
		}
	case info.Mode().IsRegular() && hdr.Linkname == "":
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cpio: failed to write %s %s to archive :%w", info.Name(), info.Mode(), err)
		}

		defer file.Close()
		if _, err = io.Copy(archive, file); err != nil {
			return fmt.Errorf("cpio: failed to write %s %s to archive :%w", info.Name(), info.Mode(), err)
		}
	}

	return nil
}
