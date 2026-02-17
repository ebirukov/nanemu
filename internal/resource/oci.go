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
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const DockerRegistryURL = "registry-1.docker.io"

type Image struct {
	raw      string
	Repo     string
	Tag      string
	Name     string
	Registry string
}

func (i *Image) String() string {
	return i.raw
}

func parseImage(uri *url.URL) *Image {
	var image Image
	imageRef := uri.Path[1:]
	image.Registry = DockerRegistryURL
	if uri.Host != "" {
		image.Registry = uri.Host
	}

	image.raw = imageRef
	parts := strings.Split(imageRef, ":")
	image.Repo = parts[0]
	image.Tag = "latest"
	if len(parts) > 1 {
		image.Tag = parts[1]
	}

	return &image
}

type DockerResolver struct {
	ociResolver *OCIRegistryResolver
}

func NewDockerResolver(arch string) *DockerResolver {
	return &DockerResolver{NewOCIResolver(arch)}
}

func (r *DockerResolver) ByURI(uri *url.URL) (string, error) {
	s := strings.Split(uri.Path[1:], "/")
	if len(s) == 1 {
		uri.Path = uri.Path[:1] + "library/" + uri.Path[1:]
	}

	return r.ociResolver.ByURI(uri)
}

type OCIRegistryResolver struct {
	arch string
}

func NewOCIResolver(arch string) *OCIRegistryResolver {
	return &OCIRegistryResolver{
		arch: arch,
	}
}

func (r *OCIRegistryResolver) ByURI(uri *url.URL) (string, error) {
	image := parseImage(uri)
	downloadPath := filepath.Join(fs.CfgDir(), "docker-image", image.Repo, image.Tag, r.arch)
	if _, err := os.Stat(downloadPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("downloading docker image:", image.String())
			if err := Download(image, r.arch, downloadPath); err != nil {
				fmt.Printf("can't download image from %s: %s\n", uri, err)

				return "", nil
			}

			return downloadPath, nil
		}

		return "", err
	}

	return downloadPath, nil
}

func Download(image *Image, arch, path string) error {
	repoCli := repo.NewClient(image.Repo, "https://"+image.Registry+"/v2")

	var manifest spec.Manifest

	if err := downloadManifest(repoCli, image.Tag, arch, &manifest); err != nil {
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
