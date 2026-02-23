package resource

import (
	"fmt"
	"github.com/ebirukov/nanemu/fs"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

func DownloadURL(uri *url.URL) (Bundle, error) {
	downloadPath := filepath.Join(fs.CfgDir(), "download", uri.Host, filepath.Dir(uri.Path))
	os.MkdirAll(downloadPath, 0755)

	bundle := Bundle{
		ContentPath: downloadPath,
		Type:        "web",
	}

	fName := filepath.Join(downloadPath, filepath.Base(uri.Path))

	if _, err := os.Stat(fName); err != nil {
		if os.IsNotExist(err) {
			fw, err := os.OpenFile(fName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
			if err != nil {
				return bundle, fmt.Errorf("can't create file %s: %w", fName, err)
			}

			defer fw.Close()

			fmt.Printf("downloading from %s to %s\n", uri.String(), downloadPath)
			if err := download(uri.String(), func(r io.Reader) error {
				_, err := io.Copy(fw, r)

				return err
			}); err != nil {
				return bundle, fmt.Errorf("can't download %s: %w", uri, err)
			}

			return bundle, err
		}
		if err != nil {
			return bundle, fmt.Errorf("can't stat %s: %w", fName, err)
		}
	}

	return bundle, nil
}

func download(url string, fetch func(io.Reader) error) error {
	httpCli := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("failed req to %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d; URL: %s, req Headers: %s; resp Headers: %s", resp.StatusCode, req.URL, req.Header, resp.Header)
	}

	defer resp.Body.Close()

	return fetch(resp.Body)
}
