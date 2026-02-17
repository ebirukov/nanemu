package diskimg

import (
	"fmt"
	"github.com/ebirukov/nanemu/pkg/cpio"
	"github.com/ebirukov/nanemu/pkg/ext4"
	"os"
	"runtime"
)

var (
	defaultPermBitsMask = map[string]int{
		"windows": 0o110,
		"linux":   0o000,
		"darwin":  0o000,
	}
)

type ImageFile struct {
	size int64
	file *os.File
}

func CreateHardDiskImage(rootFSPath string) (*ImageFile, error) {
	var imageFile ImageFile

	const diskTmplName = "hd.img"

	if !isExist(diskTmplName) {
		img, err := ext4.CreateDiskImage(diskTmplName, 512)
		if err != nil {
			return nil, fmt.Errorf("can't create hard disk image from %s: %w", rootFSPath, err)
		}

		defer img.Close()

		if imageFile.size, err = ext4.CopyFrom(rootFSPath, img, defaultPermBitsMask[runtime.GOOS]); err != nil {
			return nil, fmt.Errorf("can't copy files to hard disk image from %s: %w", rootFSPath, err)
		}
	}

	var err error

	if imageFile.file, err = os.Open(diskTmplName); err != nil {
		return nil, fmt.Errorf("can't read hard disk image %s: %w", diskTmplName, err)
	}

	return &imageFile, nil
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CreateInitRDImage(root string) (*ImageFile, error) {
	initrdFile, err := os.CreateTemp(".", "initramfs.cpio")
	if err != nil {
		return nil, fmt.Errorf("could not create file initramfs.cpio: %v", err)
	}

	if err = cpio.Create(initrdFile, root, defaultPermBitsMask[runtime.GOOS]); err != nil {
		return nil, fmt.Errorf("could not create cpio fs %s: %v", initrdFile.Name(), err)
	}

	info, err := initrdFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("could read stat %s: %v", initrdFile.Name(), err)
	}

	return &ImageFile{
		size: info.Size(),
		file: initrdFile,
	}, nil
}

func (f *ImageFile) Close() error {
	return f.file.Close()
}

func (f *ImageFile) Remove() error {
	return os.Remove(f.file.Name())
}

func (f *ImageFile) Path() string {
	return f.file.Name()
}

func (f *ImageFile) Size() int {
	return int(f.size)
}
