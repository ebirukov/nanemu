package tar

import (
	"archive/tar"
	"io"
	"os"
)

func IsTar(f *os.File) bool {
	tr := tar.NewReader(f)
	if _, err := tr.Next(); err != nil && err != io.EOF {
		return false
	}

	return true
}
