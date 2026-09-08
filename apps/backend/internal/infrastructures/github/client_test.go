package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL, "owner/repo", "token-123", server.Client())
}

func TestListReleasesFiltersDraftsAndParsesAssets(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "20" || r.URL.Query().Get("page") != "1" {
			t.Errorf("unexpected pagination params %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Errorf("missing bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"tag_name": "mobile-v1.2.0",
				"name": "Koyomi Gal Android 1.2.0 (build 42)",
				"body": "release notes",
				"draft": false,
				"prerelease": false,
				"created_at": "2026-09-01T10:00:00Z",
				"published_at": "2026-09-01T10:05:00Z",
				"assets": [
					{
						"name": "koyomi-gal-1.2.0-42-abcdef0.apk",
						"size": 12345678,
						"digest": "sha256:0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
						"browser_download_url": "https://github.com/owner/repo/releases/download/mobile-v1.2.0/koyomi-gal-1.2.0-42-abcdef0.apk",
						"updated_at": "2026-09-01T10:05:00Z"
					},
					{
						"name": "koyomi-gal-1.2.0-42-abcdef0.apk.sha256",
						"size": 96,
						"browser_download_url": "https://github.com/owner/repo/releases/download/mobile-v1.2.0/koyomi-gal-1.2.0-42-abcdef0.apk.sha256",
						"updated_at": "2026-09-01T10:05:00Z"
					}
				]
			},
			{
				"tag_name": "untagged-deadbeef",
				"name": "draft release",
				"draft": true,
				"assets": []
			}
		]`))
	}))

	releases, err := client.ListReleases(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 published release, got %d", len(releases))
	}
	release := releases[0]
	if release.TagName != "mobile-v1.2.0" || release.Prerelease || release.Draft {
		t.Fatalf("unexpected release metadata: %+v", release)
	}
	apk := release.FindAPKAsset()
	if apk == nil {
		t.Fatal("expected an apk asset")
	}
	if apk.Size != 12345678 {
		t.Errorf("unexpected apk size %d", apk.Size)
	}
	if apk.DownloadURL != "https://github.com/owner/repo/releases/download/mobile-v1.2.0/koyomi-gal-1.2.0-42-abcdef0.apk" {
		t.Errorf("unexpected download url %s", apk.DownloadURL)
	}
}

func TestListReleasesUpstreamErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		err    error
	}{
		{name: "rate limited", status: http.StatusTooManyRequests, err: ErrRateLimited},
		{name: "forbidden", status: http.StatusForbidden, err: ErrUnauthorized},
		{name: "not found", status: http.StatusNotFound, err: ErrReleaseNotFound},
		{name: "server error", status: http.StatusInternalServerError, err: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			_, err := client.ListReleases(context.Background(), 1, 20)
			if test.err == nil {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if !errors.Is(err, test.err) {
				t.Fatalf("expected %v, got %v", test.err, err)
			}
		})
	}
}

func TestParseAssetSHA256(t *testing.T) {
	valid := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	upper := "sha256:0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
	tests := []struct {
		name   string
		digest string
		want   string
		ok     bool
	}{
		{name: "valid", digest: valid, want: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ok: true},
		{name: "uppercase normalized", digest: upper, want: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ok: true},
		{name: "wrong algorithm", digest: "md5:abcdef", ok: false},
		{name: "empty", digest: "", ok: false},
		{name: "short hex", digest: "sha256:abcdef", ok: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseAssetSHA256(test.digest)
			if ok != test.ok || got != test.want {
				t.Fatalf("expected (%s,%v), got (%s,%v)", test.want, test.ok, got, ok)
			}
		})
	}
}
