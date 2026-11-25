package task

import (
	"testing"
)

func TestIsIPv4(t *testing.T) {
	cases := map[string]bool{
		"162.159.192.0/24": true,
		"1.1.1.1":          true,
		"2606:4700::/32":   false,
		"2606:4700:d0::":   false,
	}
	for in, want := range cases {
		if got := isIPv4(in); got != want {
			t.Fatalf("isIPv4(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFixIP(t *testing.T) {
	r := newIPRanges()
	cases := []struct {
		in, out, mask string
	}{
		{"1.1.1.1", "1.1.1.1/32", "/32"},
		{"1.1.1.1/24", "1.1.1.1/24", "/24"},
		{"2606:4700::", "2606:4700::/128", "/128"},
		{"2606:4700::/48", "2606:4700::/48", "/48"},
	}
	for _, c := range cases {
		if got := r.fixIP(c.in); got != c.out {
			t.Fatalf("fixIP(%q) = %q, want %q", c.in, got, c.out)
		}
		if r.mask != c.mask {
			t.Fatalf("fixIP(%q) mask = %q, want %q", c.in, r.mask, c.mask)
		}
	}
}

func TestGetIPRange(t *testing.T) {
	r := newIPRanges()
	r.parseCIDR("162.159.192.0/24")
	minIP, hosts := r.getIPRange()
	if minIP != 0 || hosts != 255 {
		t.Fatalf("getIPRange(/24) = (%d, %d), want (0, 255)", minIP, hosts)
	}

	r = newIPRanges()
	r.parseCIDR("162.159.192.0/25")
	minIP, hosts = r.getIPRange()
	if minIP != 0 || hosts != 127 {
		t.Fatalf("getIPRange(/25) = (%d, %d), want (0, 127)", minIP, hosts)
	}

	r = newIPRanges()
	r.parseCIDR("162.159.192.128/25")
	minIP, hosts = r.getIPRange()
	if minIP != 128 || hosts != 127 {
		t.Fatalf("getIPRange(.128/25) = (%d, %d), want (128, 127)", minIP, hosts)
	}
}

func TestChooseIPv4Single(t *testing.T) {
	r := newIPRanges()
	r.parseCIDR("1.1.1.1")
	r.chooseIPv4()
	if len(r.ips) != 1 {
		t.Fatalf("single IPv4 should produce 1 ip, got %d", len(r.ips))
	}
	if got := r.ips[0].IP.String(); got != "1.1.1.1" {
		t.Fatalf("ip = %q, want 1.1.1.1", got)
	}
}

func TestChooseIPv4CIDR(t *testing.T) {
	r := newIPRanges()
	r.parseCIDR("10.0.0.0/30")
	r.chooseIPv4()
	// 10.0.0.0 .. 10.0.0.3
	if len(r.ips) != 4 {
		t.Fatalf("/30 should produce 4 ips, got %d", len(r.ips))
	}
	want := []string{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"}
	for i, ip := range r.ips {
		if got := ip.IP.String(); got != want[i] {
			t.Fatalf("ips[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestLoadIPRangesBuiltin(t *testing.T) {
	oldText, oldFile, oldV6 := IPText, IPFile, IPv6Mode
	defer func() { IPText, IPFile, IPv6Mode = oldText, oldFile, oldV6 }()
	IPText, IPFile = "", ""

	IPv6Mode = false
	ips := loadIPRanges()
	// 8 IPv4 /24 ranges = 2048 addresses.
	if len(ips) != 2048 {
		t.Fatalf("builtin IPv4 = %d ips, want 2048", len(ips))
	}

	IPv6Mode = true
	ips = loadIPRanges()
	if len(ips) == 0 {
		t.Fatalf("builtin IPv6 should produce addresses")
	}
}
