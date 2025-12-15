package initrd

import (
	"fmt"
	"github.com/ebirukov/nanemu/pkg/cpio"
	"os"
)

type ImageFile struct {
	info os.FileInfo
	file *os.File
}

func CreateImage(name string, root string, execPermBits int) (*ImageFile, error) {
	initrdFile, err := os.CreateTemp(".", name)
	if err != nil {
		return nil, fmt.Errorf("could not create file initramfs.cpio: %v", err)
	}

	if err = cpio.Create(initrdFile, root, execPermBits); err != nil {
		return nil, fmt.Errorf("could not create cpio fs %s: %v", initrdFile.Name(), err)
	}

	info, err := os.Stat(initrdFile.Name())
	if err != nil {
		return nil, fmt.Errorf("could read stat %s: %v", initrdFile.Name(), err)
	}

	return &ImageFile{
		info: info,
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
	return int(f.info.Size())
}
