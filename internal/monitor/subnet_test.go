package monitor

import "testing"

func TestSubnetPrefix(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		v4   int
		want string
	}{
		{"v4_default_24", "176.59.35.168", 24, "176.59.35.0/24"},
		{"v4_16", "176.59.35.168", 16, "176.59.0.0/16"},
		{"v4_20", "176.59.45.158", 20, "176.59.32.0/20"},
		{"v4_32_exact", "1.2.3.4", 32, "1.2.3.4/32"},
		{"v6_returned_as_is", "2001:db8:abcd:1234::1", 24, "2001:db8:abcd:1234::1"},
		{"v6_loopback_as_is", "::1", 24, "::1"},
		{"invalid", "not-an-ip", 24, "not-an-ip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := subnetPrefix(tc.ip, tc.v4)
			if got != tc.want {
				t.Errorf("subnetPrefix(%q, %d) = %q, want %q", tc.ip, tc.v4, got, tc.want)
			}
		})
	}
}
