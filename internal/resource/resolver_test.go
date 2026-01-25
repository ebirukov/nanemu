package resource

import (
	"net/url"
	"testing"
)

type mockFetcher struct {
	got *url.URL
	ret string
	err error
}

func (m *mockFetcher) ByURI(u *url.URL) (string, error) {
	m.got = u
	return m.ret, m.err
}

func TestFetchPath_ParseOnly(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "windows path is not URI",
			input:      "C:\\a\\b.txt",
			wantScheme: "file",
			wantPath:   "C:\\a\\b.txt",
		},
		{
			name:       "windows file uri",
			input:      "file:///C:/a/b.txt",
			wantScheme: "file",
			wantPath:   "/C:/a/b.txt",
		},
		/*		{
					name:       "unix absolute path treated as file",
					input:      "/a/b/c.txt",
					wantScheme: "file",
					wantPath:   "/a/b/c.txt",
				},
				{
					name:       "relative path treated as file",
					input:      "a/b/c.txt",
					wantScheme: "file",
					wantPath:   "a/b/c.txt",
				},*/
		{
			name:       "file uri unix style",
			input:      "file:///a/b/c.txt",
			wantScheme: "file",
			wantPath:   "/a/b/c.txt",
		},
		{
			name:       "http uri",
			input:      "http://example.com/a/b",
			wantScheme: "http",
			wantPath:   "/a/b",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mf := &mockFetcher{ret: "ok"}

			fetchers := BySchemaFetchers{
				tc.wantScheme: mf,
			}

			_, err := fetchers.FetchPath(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mf.got == nil {
				t.Fatalf("fetcher was not called")
			}

			if mf.got.Scheme != tc.wantScheme {
				t.Fatalf("scheme: expected %q, got %q", tc.wantScheme, mf.got.Scheme)
			}

			if mf.got.Path != tc.wantPath {
				t.Fatalf("path: expected %q, got %q", tc.wantPath, mf.got.Path)
			}
		})
	}
}
