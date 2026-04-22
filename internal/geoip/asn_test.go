package geoip

import (
	"testing"
)

func TestNopResolver_AlwaysReturnsMiss(t *testing.T) {
	r := NopResolver{}
	info, ok := r.Lookup("8.8.8.8")
	if ok {
		t.Errorf("NopResolver.Lookup returned ok=true, want false")
	}
	if info.Number != 0 {
		t.Errorf("NopResolver.Lookup returned Number=%d, want 0", info.Number)
	}
	if info.Org != "" {
		t.Errorf("NopResolver.Lookup returned Org=%q, want empty", info.Org)
	}
}

func TestNewDBResolver_FileNotFound(t *testing.T) {
	_, err := NewDBResolver("/nonexistent/path/GeoLite2-ASN.mmdb")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestNewDBResolver_EmptyPath(t *testing.T) {
	_, err := NewDBResolver("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func TestDBResolver_Reload_NonexistentFile_ReturnsError(t *testing.T) {
	d := &DBResolver{}
	if err := d.Reload("/nonexistent/path/foo.mmdb"); err == nil {
		t.Fatal("expected error on Reload with nonexistent path, got nil")
	}
}

func TestDBResolver_Reload_EmptyPath_ReturnsError(t *testing.T) {
	d := &DBResolver{}
	if err := d.Reload(""); err == nil {
		t.Fatal("expected error on Reload with empty path, got nil")
	}
}
