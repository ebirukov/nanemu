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

type HTTPFetcher struct {
	arch string
}

func NewHTTPFetcher(arch string) *HTTPFetcher {
	return &HTTPFetcher{
		arch: arch,
	}
}

func (r *HTTPFetcher) ByURI(uri *url.URL) (string, error) {
	downloadPath := filepath.Join(fs.CfgDir(), "kernel", r.arch)
	os.Mkdir(downloadPath, 0755)

	fw, err := os.OpenFile(filepath.Join(downloadPath, "vmlinuz"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("can't create file %s: %w", fw.Name(), err)
	}

	defer fw.Close()

	if err := download(uri.String(), func(r io.Reader) error {
		_, err := io.Copy(fw, r)

		return err
	}); err != nil {
		return "", fmt.Errorf("can't download %s: %w", uri, err)
	}

	return filepath.Join(downloadPath, "vmlinuz"), nil
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
