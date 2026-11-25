package task

import (
	"net"
	"testing"
)

// TestWarpingLoopback drives the full measurement path against a local UDP
// responder that echoes a 92-byte WireGuard handshake response, so no packet
// leaves the loopback interface.
func TestWarpingLoopback(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2408})
	if err != nil {
		t.Skipf("cannot bind loopback: %v", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 2048)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n == 0 {
				continue
			}
			resp := make([]byte, 92)
			if _, err := conn.WriteToUDP(resp, addr); err != nil {
				return
			}
		}
	}()

	oldTimes := PingTimes
	PingTimes = 2
	defer func() { PingTimes = oldTimes }()

	ip := &UDPAddr{IP: &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}, Port: 2408}
	w := &Warping{}
	received, total := w.warping(ip)
	if received != PingTimes {
		t.Fatalf("received = %d, want %d", received, PingTimes)
	}
	if total <= 0 {
		t.Fatalf("total delay = %v, want > 0", total)
	}
}

func TestFullAddress(t *testing.T) {
	cases := []struct {
		ip   string
		port int
		want string
	}{
		{"1.2.3.4", 2408, "1.2.3.4:2408"},
		{"2606:4700:d0::", 2408, "[2606:4700:d0::]:2408"},
	}
	for _, c := range cases {
		u := &UDPAddr{IP: &net.IPAddr{IP: net.ParseIP(c.ip)}, Port: c.port}
		if got := u.FullAddress(); got != c.want {
			t.Fatalf("FullAddress(%s) = %q, want %q", c.ip, got, c.want)
		}
	}
}

func TestGenerateSingleIPAddr(t *testing.T) {
	ips := []*net.IPAddr{{IP: net.IPv4(1, 1, 1, 1)}, {IP: net.IPv4(2, 2, 2, 2)}}
	addrs := generateSingleIPAddr(ips, 2408)
	if len(addrs) != 2 {
		t.Fatalf("len = %d, want 2", len(addrs))
	}
	if addrs[0].Port != 2408 || addrs[1].Port != 2408 {
		t.Fatalf("ports = %d,%d want 2408,2408", addrs[0].Port, addrs[1].Port)
	}
	if addrs[0].IP.IP.String() != "1.1.1.1" {
		t.Fatalf("first ip = %v", addrs[0].IP.IP)
	}
}

func TestPortsHaveNoDuplicates(t *testing.T) {
	seen := map[int]bool{}
	for _, p := range ports {
		if seen[p] {
			t.Fatalf("duplicate port %d", p)
		}
		seen[p] = true
	}
	if len(ports) != 54 {
		t.Fatalf("port count = %d, want 54", len(ports))
	}
	if _, ok := seen[2408]; !ok {
		t.Fatalf("port 2408 missing")
	}
}

func TestDefaultHandshakePacket(t *testing.T) {
	if len(warpHandshakePacket) != 148 {
		t.Fatalf("handshake packet len = %d, want 148", len(warpHandshakePacket))
	}
	if warpHandshakePacket[0] != 1 {
		t.Fatalf("packet type = %d, want 1", warpHandshakePacket[0])
	}
	if warpHandshakePacket[1] != 0x3c || warpHandshakePacket[2] != 0xbd || warpHandshakePacket[3] != 0xaf {
		t.Fatalf("reserved bytes = %x, want 3c bd af", warpHandshakePacket[1:4])
	}
}

func TestQuickModeSwitch(t *testing.T) {
	oldQuick, oldAll := QuickMode, ScanAllPort
	defer func() { QuickMode, ScanAllPort = oldQuick, oldAll }()
	ScanAllPort = false

	// 内置 IPv4 = 2048 IP × 54 端口 = 110592 条
	full := 2048 * len(ports)

	// -q（默认 true）：截断到 1000 条
	QuickMode = true
	if got := len(loadWarpIPRanges()); got != quickModeMaxIpNum {
		t.Fatalf("quick mode = %d, want %d", got, quickModeMaxIpNum)
	}

	// -q=false：扫所有，不截断
	QuickMode = false
	if got := len(loadWarpIPRanges()); got != full {
		t.Fatalf("full mode = %d, want %d", got, full)
	}
}
