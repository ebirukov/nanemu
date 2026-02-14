package resource

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type Fetcher interface {
	ByURI(uri *url.URL) (string, error)
}
type FetcherFn func(*url.URL) (string, error)

func (r FetcherFn) ByURI(u *url.URL) (string, error) {
	return r(u)
}

type BySchemaFetchers map[string]Fetcher

func (r BySchemaFetchers) AddFetcher(schema string, fetcher Fetcher) {
	r[schema] = fetcher
}

var DefaultFetcher = BySchemaFetchers{
	"file": FetcherFn(FetchFilePath),
}

func (r BySchemaFetchers) FetchPath(uri string) (string, error) {
	resourceUri, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("can't parse URI: %w", err)
	}

	if resourceUri.Path == "" && resourceUri.Opaque != "" {
		resourceUri.Path = uri
		resourceUri.Scheme = "file"
	}

	if resourceUri.Path == "" {
		return "", fmt.Errorf("empty path of URI %s", uri)
	}

	schema := resourceUri.Scheme
	if schema == "" {
		schema = "file"
	}

	fetcher, ok := r[schema]
	if !ok {
		return "", fmt.Errorf("unsupported schema '%s' of URI %s\n", schema, uri)
	}

	path, err := fetcher.ByURI(resourceUri)
	if err != nil {
		return "", fmt.Errorf("can't fetch URI: %w", err)
	}

	return path, nil
}

func FetchFilePath(uri *url.URL) (string, error) {
	path := uri.Path

	if absPath, err := filepath.Abs(path); err == nil {
		path = absPath
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("can't find file path %s: %w", path, os.ErrNotExist)
	}

	if info.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return "", fmt.Errorf("can't read directory %s: %w", path, err)
		}

		if len(files) == 0 {
			return "", fmt.Errorf("directory %s has no files", path)
		}

		return filepath.Join(path, files[0].Name()), nil
	}

	return path, nil
}
