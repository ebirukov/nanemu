package repo

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/opencontainers/go-digest"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
)

type Client struct {
	httpCli     *http.Client
	registryURL string
	repoName    string
	authToken   string
}

var UnsecureTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

// NewClient создает registry api клиента для доступа репозиторию repoName
func NewClient(repoName string, registryURL string) *Client {
	return &Client{
		repoName:    repoName,
		registryURL: registryURL,
		httpCli: &http.Client{
			Transport: UnsecureTransport,
		},
	}
}

// GetManifest загружает json (index или manifest) из CAS реестра по ref (hash или имя репозитория)
func (cl *Client) GetManifest(ref string, fetch func(string, io.Reader) error) error {
	path, _ := neturl.JoinPath(cl.registryURL, cl.repoName, "manifests", ref)
	resp, err := cl.HttpGet(path)
	if err != nil {
		return fmt.Errorf("can't get manifest: %w", err)
	}

	defer resp.Body.Close()

	return fetch(resp.Header.Get("Content-Type"), resp.Body)
}

// DownloadBlob загружает из CAS реестра образов содержимое подписанного слоя
func (cl *Client) DownloadBlob(digest digest.Digest, fetch func(io.Reader) error) error {
	rURL, _ := neturl.JoinPath(cl.registryURL, cl.repoName, "blobs", digest.String())
	resp, err := cl.HttpGet(rURL)
	if err != nil {
		return fmt.Errorf("failed get layer %s: %w", digest.Hex(), err)
	}

	defer resp.Body.Close()

	return fetch(resp.Body)
}

func (cl *Client) HttpGet(url string, acceptedMediaTypes ...string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+cl.authToken)

	if len(acceptedMediaTypes) > 0 {
		req.Header.Add("Accept", strings.Join(acceptedMediaTypes, ","))
	}

	resp, err := cl.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed req to %s: %w", url, err)
	}

	if cl.authToken == "" && resp.StatusCode == http.StatusUnauthorized {
		cl.authToken, err = cl.getAuthToken(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to get auth token: %v", err)
		}

		return cl.HttpGet(url, acceptedMediaTypes...)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d; URL: %s", resp.StatusCode, req.URL)
	}

	return resp, nil
}

// getAuthToken получает публичный токен авторизации
func (cl *Client) getAuthToken(resp *http.Response) (string, error) {
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return "", fmt.Errorf("registry requires auth but no challenge provided")
	}

	url, err := parseAuthURL(authHeader)
	if err != nil {
		return "", fmt.Errorf("failed to parse auth url from header %s: %w", authHeader, err)
	}

	resp, err = cl.httpCli.Get(url.String())
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed %d", resp.StatusCode)
	}

	var tokenResp = struct {
		Token string `json:"token"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %v", err)
	}

	return tokenResp.Token, nil
}

// парсит заголовок вида Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/busybox:pull"
// в URL вида https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/busybox:pull
func parseAuthURL(challengeBasedHeader string) (*neturl.URL, error) {
	header, _ := strings.CutPrefix(challengeBasedHeader, "Bearer ")
	q := neturl.Values{}
	var realm string
	for _, part := range strings.Split(header, ",") {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "realm" {
			realm = strings.Trim(kv[1], "\"")
		} else {
			q.Set(kv[0], strings.Trim(kv[1], "\""))
		}
	}

	return neturl.Parse(realm + "?" + q.Encode())
}
