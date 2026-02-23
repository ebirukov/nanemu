package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// UnpackTo распаковывает архив в директорию destPath
func UnpackTo(src io.Reader, destPath string) error {
	tr := tar.NewReader(src)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // конец архива
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		// безопасный путь
		target, err := safeJoin(destPath, header.Name)
		if err != nil {
			return err
		}

		fType := header.Typeflag
		if runtime.GOOS == "windows" && fType == tar.TypeSymlink {
			fType = tar.TypeLink
		}

		switch fType {
		case tar.TypeDir: // директория
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}

		case tar.TypeReg: // обычный файл
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			os.Remove(target)
			fw, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("can't create file %s: %w", target, err)
			}
			if _, err := io.Copy(fw, tr); err != nil {
				fw.Close()
				return err
			}
			fw.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("create dir for symlink %s: %w", target, err)
			}

			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", header.Linkname, err)
			}

		case tar.TypeLink:
			// жёсткая ссылка
			linkTarget, err := safeJoin(destPath, header.Linkname)
			if err != nil {
				return err
			}

			os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}

		default:
			fmt.Printf("skip unsupported entry: %s\n", header.Name)
		}
	}

	return nil
}

// safeJoin защищает от tar path traversal атак
func safeJoin(base, rel string) (string, error) {
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid tar path: %s", rel)
	}
	dst := filepath.Join(base, rel)
	return dst, nil
}
