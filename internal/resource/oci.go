package resource

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/ebirukov/nanemu/fs"
	"github.com/ebirukov/nanemu/pkg/repo"
	"github.com/ebirukov/nanemu/pkg/tar"
	spec "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"strings"
)

const DockerRegistryURL = "https://registry-1.docker.io/v2"

type OCIRegistryResolver struct {
	arch string
}

func NewResolver(arch string) *OCIRegistryResolver {
	return &OCIRegistryResolver{
		arch: arch,
	}
}

func (r *OCIRegistryResolver) ByURI(uri *url.URL) (string, error) {
	downloadPath := filepath.Join(fs.CfgDir(), "kernel", r.arch)
	if err := Download(uri.Path[1:], r.arch, downloadPath); err != nil {
		log.Printf("can't download kernel: %s/n", err)

		return "", nil
	}

	return DefaultFetcher.FetchPath("file://" + downloadPath)
}

func Download(image string, arch, path string) error {
	parts := strings.Split(image, ":")
	imageRepo := parts[0]
	ref := "latest"
	if len(parts) > 1 {
		ref = parts[1]
	}

	repoCli := repo.NewClient(imageRepo, DockerRegistryURL)

	var manifest spec.Manifest

	if err := downloadManifest(repoCli, ref, arch, &manifest); err != nil {
		return fmt.Errorf("can't get manifest: %v", err)
	}

	for i, layerDesc := range manifest.Layers {
		fetchFn := func(r io.Reader) error {
			return storeBlob(r, layerDesc, path)
		}

		if err := repoCli.DownloadBlob(layerDesc.Digest, fetchFn); err != nil {
			return fmt.Errorf("can't fetch blob layer %d: %w", i, err)
		}
	}

	return nil
}

func downloadManifest(cl *repo.Client, ref, arch string, manifest *spec.Manifest) error {
	fetchManifestFn := func(mediaType string, r io.Reader) error {
		switch mediaType {
		case spec.MediaTypeImageIndex: // список манифестов для разных ОС и аппаратных платформ
			var idx *spec.Index
			if err := json.NewDecoder(r).Decode(&idx); err != nil {
				return err
			}

			for _, m := range idx.Manifests {
				if m.Platform != nil && m.Platform.OS == "linux" && m.Platform.Architecture == arch {
					return downloadManifest(cl, m.Digest.String(), arch, manifest)
				}
			}

			return fmt.Errorf("can't find manifest in %v", idx)
		case spec.MediaTypeImageManifest:
			if err := json.NewDecoder(r).Decode(&manifest); err != nil {
				return fmt.Errorf("can't decode manifest")
			}

			return nil
		default:
			data, _ := io.ReadAll(r)
			return fmt.Errorf("unkhown manifest media type: %s; content: %s", mediaType, data)
		}
	}

	return cl.GetManifest(ref, fetchManifestFn)
}

// storeBlob содержимое сохраняет в path
func storeBlob(r io.Reader, desc spec.Descriptor, path string) error {
	var err error

	if desc.MediaType == spec.MediaTypeImageLayerGzip {
		r, err = gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("failed decompress blob %s: %w", desc.Digest.Hex(), err)
		}
	}

	switch desc.MediaType {
	case spec.MediaTypeImageLayerGzip, spec.MediaTypeImageLayer:
		if err := tar.UnpackTo(r, path); err != nil {
			return fmt.Errorf("can't unpack tar to %s: %w", path, err)
		}
	default:
		return fmt.Errorf("blob %s have unsupported media type: %s", desc.Digest, desc.MediaType)
	}

	fmt.Printf("archive (digest: %s, size: %d) saved to %s\n", desc.Digest[:16], desc.Size, path)

	return nil
}
