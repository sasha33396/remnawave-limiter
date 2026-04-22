package geoip

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildFakeArchive(t *testing.T, mmdbPayload []byte, includeMMDB bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{
		Name: "GeoLite2-ASN_20260101/README.txt",
		Mode: 0644,
		Size: int64(len("ignored")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("ignored")); err != nil {
		t.Fatal(err)
	}

	if includeMMDB {
		if err := tw.WriteHeader(&tar.Header{
			Name: "GeoLite2-ASN_20260101/GeoLite2-ASN.mmdb",
			Mode: 0644,
			Size: int64(len(mmdbPayload)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(mmdbPayload); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDownloader_Download_Success(t *testing.T) {
	fakePayload := []byte("fake-mmdb-contents")
	archive := buildFakeArchive(t, fakePayload, true)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("license_key"); got != "abc123" {
			t.Errorf("license_key = %q, want abc123", got)
		}
		if got := r.URL.Query().Get("edition_id"); got != "GeoLite2-ASN" {
			t.Errorf("edition_id = %q, want GeoLite2-ASN", got)
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(archive)
	}))
	defer ts.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "GeoLite2-ASN.mmdb")

	d := &Downloader{
		HTTPClient: ts.Client(),
		LicenseKey: "abc123",
		BaseURL:    ts.URL,
		Validate:   func(path string) error { return nil },
		Timeout:    5 * time.Second,
	}

	if err := d.Download(context.Background(), dst); err != nil {
		t.Fatalf("Download error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if !bytes.Equal(got, fakePayload) {
		t.Errorf("dst contents = %q, want %q", got, fakePayload)
	}
}

func TestDownloader_Download_HTTPError_4xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("bad key"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "GeoLite2-ASN.mmdb")

	d := &Downloader{
		HTTPClient: ts.Client(),
		LicenseKey: "wrong",
		BaseURL:    ts.URL,
		Validate:   func(string) error { return nil },
		Timeout:    5 * time.Second,
	}

	err := d.Download(context.Background(), dst)
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want containing 401", err)
	}

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("dst exists, want non-existent on download failure")
	}
}

func TestDownloader_Download_MissingMMDB(t *testing.T) {
	archive := buildFakeArchive(t, nil, false)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer ts.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "GeoLite2-ASN.mmdb")

	d := &Downloader{
		HTTPClient: ts.Client(),
		LicenseKey: "abc",
		BaseURL:    ts.URL,
		Validate:   func(string) error { return nil },
		Timeout:    5 * time.Second,
	}

	err := d.Download(context.Background(), dst)
	if err == nil {
		t.Fatal("expected error when archive has no .mmdb, got nil")
	}
}

func TestDownloader_Download_ValidateFailure_PreservesOldFile(t *testing.T) {
	archive := buildFakeArchive(t, []byte("new-contents"), true)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer ts.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "GeoLite2-ASN.mmdb")
	if err := os.WriteFile(dst, []byte("old-contents"), 0644); err != nil {
		t.Fatal(err)
	}

	d := &Downloader{
		HTTPClient: ts.Client(),
		LicenseKey: "abc",
		BaseURL:    ts.URL,
		Validate:   func(string) error { return os.ErrInvalid },
		Timeout:    5 * time.Second,
	}

	err := d.Download(context.Background(), dst)
	if err == nil {
		t.Fatal("expected error on validate failure, got nil")
	}

	got, _ := os.ReadFile(dst)
	if string(got) != "old-contents" {
		t.Errorf("dst contents = %q, want %q (old file must be preserved)", got, "old-contents")
	}
}

func TestDownloader_Download_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Write([]byte("slow"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "GeoLite2-ASN.mmdb")

	d := &Downloader{
		HTTPClient: ts.Client(),
		LicenseKey: "abc",
		BaseURL:    ts.URL,
		Validate:   func(string) error { return nil },
		Timeout:    5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := d.Download(ctx, dst)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}
