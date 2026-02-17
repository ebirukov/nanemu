package ext4

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

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

func CopyFrom(srcDir string, img *ext4fs.Image) (total int64, err error) {
	// храним inode созданных директорий
	inodes := map[string]uint32{
		"/": ext4fs.RootInode,
	}

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
		case info.Mode().IsRegular(): // ← обычный файл
			data, err := os.ReadFile(hostPath)
			if err != nil {
				return err
			}

			_, err = img.CreateFile(parentInode, d.Name(), data, mode, 0, 0)
			if err != nil {
				return err
			}

			total += info.Size()

		default:
			return nil
		}

		return nil
	})

	return

}
