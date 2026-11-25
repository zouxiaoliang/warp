package utils

import (
	"net"
	"testing"
	"time"
)

func mustUDP(t *testing.T, s string) *net.UDPAddr {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		t.Fatalf("resolve %q: %v", s, err)
	}
	return addr
}

func TestToString(t *testing.T) {
	cf := CloudflareIPData{
		PingData: &PingData{
			IP:       mustUDP(t, "162.159.195.222:5279"),
			Sended:   10,
			Received: 10,
			Delay:    224981234 * time.Nanosecond,
		},
	}
	got := cf.toString()
	want := []string{"162.159.195.222:5279", "0%", "224.98"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("toString()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGetLossRate(t *testing.T) {
	cf := CloudflareIPData{PingData: &PingData{Sended: 10, Received: 7}}
	if got := cf.getLossRate(); got != 0.3 {
		t.Fatalf("lossRate = %v, want 0.3", got)
	}
	// Cached value must be returned on the second call.
	if got := cf.getLossRate(); got != 0.3 {
		t.Fatalf("cached lossRate = %v, want 0.3", got)
	}
}

func TestFilterDelay(t *testing.T) {
	oldMin, oldMax := InputMinDelay, InputMaxDelay
	defer func() { InputMinDelay, InputMaxDelay = oldMin, oldMax }()

	s := PingDelaySet{
		{PingData: &PingData{Delay: 50 * time.Millisecond}},
		{PingData: &PingData{Delay: 200 * time.Millisecond}},
		{PingData: &PingData{Delay: 400 * time.Millisecond}},
		{PingData: &PingData{Delay: 100 * time.Millisecond}},
	}
	InputMinDelay, InputMaxDelay = 100*time.Millisecond, 300*time.Millisecond
	got := s.FilterDelay()
	if len(got) != 1 || got[0].Delay != 200*time.Millisecond {
		t.Fatalf("FilterDelay = %+v, want single 200ms entry", got)
	}
}

func TestFilterLossRate(t *testing.T) {
	old := InputMaxLossRate
	defer func() { InputMaxLossRate = old }()

	s := PingDelaySet{
		{PingData: &PingData{Sended: 10, Received: 10}},
		{PingData: &PingData{Sended: 10, Received: 8}},
		{PingData: &PingData{Sended: 10, Received: 5}},
	}
	InputMaxLossRate = 0.1
	got := s.FilterLossRate()
	if len(got) != 1 || got[0].getLossRate() != 0 {
		t.Fatalf("FilterLossRate = %+v, want single 0%% entry", got)
	}
}

func TestSort(t *testing.T) {
	s := PingDelaySet{
		{PingData: &PingData{Sended: 10, Received: 5, Delay: 100 * time.Millisecond}},
		{PingData: &PingData{Sended: 10, Received: 10, Delay: 300 * time.Millisecond}},
		{PingData: &PingData{Sended: 10, Received: 10, Delay: 100 * time.Millisecond}},
	}
	// Loss rate ascending, then delay ascending.
	if !s.Less(2, 1) {
		t.Fatalf("Less(2,1) should be true (same loss, lower delay first)")
	}
	if !s.Less(1, 0) {
		t.Fatalf("Less(1,0) should be true (lower loss first)")
	}
}
