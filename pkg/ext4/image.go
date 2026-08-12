package ext4

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/ebirukov/nanemu/pkg/fsutil"

	"github.com/pilat/go-ext4fs"
)

func CreateDiskImage(path string, sizeMB int) (*ext4fs.Image, error) {
	img, err := ext4fs.New(ext4fs.WithImagePath(path), ext4fs.WithSizeInMB(sizeMB))
	if err != nil {
		return nil, fmt.Errorf("can't create ext4 disk image %s: %w", path, err)
	}

	img.DeleteDirectory(ext4fs.RootInode, "lost+found")
	_, createDirErr := img.CreateDirectory(ext4fs.RootInode, "dev", 0755, 0, 0)

	if err := errors.Join(createDirErr, img.Save()); err != nil {
		return nil, fmt.Errorf("can't create dev directory on ext4 disk image %s: %w", path, err)
	}

	return img, nil
}

func CopyFrom(srcDir string, img *ext4fs.Image, execPermBits int) (total int64, err error) {
	// храним inode созданных директорий
	inodes := map[string]uint32{
		"/": ext4fs.RootInode,
	}
	// ключ (inode) -> созданный inode, для детекции хардлинков
	hardlinks := map[fsutil.Inode]uint32{}

	defer func() {
		err = img.Save()
	}()

	err = filepath.WalkDir(srcDir, func(hostPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, hostPath)
		if err != nil {
			return err
		}

		if d.IsDir() && rel == "." {
			return nil
		}

		rel = filepath.ToSlash(rel)
		fullPath := "/" + rel
		parentPath := path.Dir(fullPath)

		parentInode, ok := inodes[parentPath]
		if !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		mode := uint16(info.Mode().Perm())

		switch {
		case d.IsDir():
			inode, err := img.CreateDirectory(parentInode, d.Name(), mode, 0, 0)
			if err != nil {
				return err
			}
			inodes[fullPath] = inode

		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(hostPath)
			if err != nil {
				return err
			}
			target = filepath.ToSlash(target)

			_, err = img.CreateSymlink(parentInode, d.Name(), target, 0, 0)
			if err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if key, ok := fsutil.InodeKeyOf(info); ok {
				if targetInode, seen := hardlinks[key]; seen {
					if err := img.Link(parentInode, d.Name(), targetInode); err != nil {
						return err
					}
					return nil
				}
			}

			mode = restoreExecMode(hostPath, execPermBits, info)

			data, err := os.ReadFile(hostPath)
			if err != nil {
				return err
			}

			fileInode, err := img.CreateFile(parentInode, d.Name(), data, mode, 0, 0)
			if err != nil {
				return err
			}

			if key, ok := fsutil.InodeKeyOf(info); ok {
				hardlinks[key] = fileInode
			}

			total += info.Size()

		default:
			return nil
		}

		return nil
	})

	return

}

func restoreExecMode(hostPath string, execPermBits int, info fs.FileInfo) uint16 {
	mode := uint16(info.Mode().Perm())

	if execPermBits > 0 {
		if isExec, _ := fsutil.IsExecutableFile(hostPath); isExec {
			fmt.Printf("file %s permision mode flags %v", info.Name(), info.Mode())
			execMode := info.Mode() | fs.FileMode(execPermBits)
			fmt.Printf(" was modified to %v\n", execMode)
			mode = uint16(execMode.Perm())
		}
	}
	return mode
}

func extractTarToExt4(r io.Reader, img *ext4fs.Image, inodes map[string]uint32) (int64, error) {
	tr := tar.NewReader(r)

	var total int64
	tarInodes := make(map[string]uint32)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}

		name := path.Clean("/" + hdr.Name)
		parent := path.Dir(name)

		parentInode, ok := inodes[parent]
		if !ok {
			return total, err
		}

		mode := uint16(hdr.Mode)

		switch hdr.Typeflag {

		case tar.TypeDir:
			inode, err := img.CreateDirectory(parentInode, path.Base(name), mode, 0, 0)
			if err != nil {
				return total, err
			}
			tarInodes[hdr.Name] = inode

		case tar.TypeReg:
			data, err := io.ReadAll(tr)
			if err != nil {
				return total, err
			}

			inode, err := img.CreateFile(parentInode, path.Base(name), data, mode, 0, 0)
			if err != nil {
				return total, err
			}

			tarInodes[hdr.Name] = inode
			total += hdr.Size

		case tar.TypeSymlink:
			_, err := img.CreateSymlink(parentInode, path.Base(name), hdr.Linkname, 0, 0)
			if err != nil {
				return total, err
			}

		case tar.TypeLink: // hard link
			/*			targetInode, ok := tarInodes[hdr.Linkname]
						if !ok {
							return total, fmt.Errorf("hardlink target not found: %s", hdr.Linkname)
						}

						_, err := img.CreateHardLink(parentInode, path.Base(name), targetInode)
						if err != nil {
							return total, err
						}*/
		}
	}

	return total, nil
}
