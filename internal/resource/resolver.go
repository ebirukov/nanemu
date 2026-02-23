package resource

import (
	"fmt"
	"net/url"
	"path/filepath"
)

type Bundle struct {
	ContentPath  string
	MetadataPath string
	Type         string
}

type Fetcher interface {
	ByURI(uri *url.URL) (Bundle, error)
}
type FetcherFn func(*url.URL) (Bundle, error)

func (r FetcherFn) ByURI(u *url.URL) (Bundle, error) {
	return r(u)
}

type BySchemaFetchers map[string]Fetcher

func (r BySchemaFetchers) AddFetcher(schema string, fetcher Fetcher) {
	r[schema] = fetcher
}

var DefaultFetcher = BySchemaFetchers{
	"https": FetcherFn(DownloadURL),
}

func (r BySchemaFetchers) FetchPath(uri string) (Bundle, error) {
	var bundle Bundle
	resourceUri, err := url.Parse(uri)
	if err != nil {
		return bundle, fmt.Errorf("can't parse URI: %w", err)
	}

	if resourceUri.Path == "" && resourceUri.Opaque != "" {
		resourceUri.Path = uri
		resourceUri.Scheme = "file"
	}

	if resourceUri.Path == "" {
		return bundle, fmt.Errorf("empty path of URI %s", uri)
	}

	schema := resourceUri.Scheme
	if schema == "" || schema == "file" {
		bundle.Type = "file"
		bundle.ContentPath = resourceUri.Path

		if absPath, err := filepath.Abs(resourceUri.Path); err == nil {
			bundle.ContentPath = absPath
		}

		return bundle, nil
	}

	fetcher, ok := r[schema]
	if !ok {
		return bundle, fmt.Errorf("unsupported schema '%s' of URI %s\n", schema, uri)
	}

	bundle, err = fetcher.ByURI(resourceUri)
	if err != nil {
		return bundle, fmt.Errorf("can't fetch URI: %w", err)
	}

	return bundle, nil
}
