package geoip

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/oschwald/maxminddb-golang"
)

type ASNInfo struct {
	Number uint32
	Org    string
}

type Resolver interface {
	Lookup(ipStr string) (ASNInfo, bool)
}

type NopResolver struct{}

func (NopResolver) Lookup(string) (ASNInfo, bool) { return ASNInfo{}, false }

type DBResolver struct {
	reader atomic.Pointer[maxminddb.Reader]
}

func NewDBResolver(path string) (*DBResolver, error) {
	if path == "" {
		return nil, fmt.Errorf("geoip: database path is empty")
	}
	r, err := maxminddb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("geoip: open %s: %w", path, err)
	}
	d := &DBResolver{}
	d.reader.Store(r)
	return d, nil
}

func (d *DBResolver) Reload(path string) error {
	if d == nil {
		return fmt.Errorf("geoip: Reload called on nil DBResolver")
	}
	if path == "" {
		return fmt.Errorf("geoip: Reload path is empty")
	}
	r, err := maxminddb.Open(path)
	if err != nil {
		return fmt.Errorf("geoip: reload open %s: %w", path, err)
	}
	old := d.reader.Swap(r)
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (d *DBResolver) Close() error {
	if d == nil {
		return nil
	}
	old := d.reader.Swap(nil)
	if old == nil {
		return nil
	}
	return old.Close()
}

func (d *DBResolver) Lookup(ipStr string) (ASNInfo, bool) {
	if d == nil {
		return ASNInfo{}, false
	}
	r := d.reader.Load()
	if r == nil {
		return ASNInfo{}, false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ASNInfo{}, false
	}
	var rec struct {
		ASN uint32 `maxminddb:"autonomous_system_number"`
		Org string `maxminddb:"autonomous_system_organization"`
	}
	if err := r.Lookup(ip, &rec); err != nil {
		return ASNInfo{}, false
	}
	if rec.ASN == 0 {
		return ASNInfo{}, false
	}
	return ASNInfo{Number: rec.ASN, Org: rec.Org}, true
}
