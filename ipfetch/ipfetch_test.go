package ipfetch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ips-v4":
			io.WriteString(w, "173.245.48.0/20\n162.158.0.0/15\n")
		case "/ips-v6":
			io.WriteString(w, "2606:4700::/32\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := URLs
	URLs = []string{srv.URL + "/ips-v4", srv.URL + "/ips-v6"}
	defer func() { URLs = old }()

	var buf bytes.Buffer
	if err := Write(&buf, srv.Client()); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"173.245.48.0/20", "162.158.0.0/15", "2606:4700::/32"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in output:\n%s", want, got)
		}
	}
	if strings.Index(got, "162.158.0.0/15") > strings.Index(got, "2606:4700::/32") {
		t.Fatalf("IPv4 ranges should precede IPv6 ranges:\n%s", got)
	}
}

func TestWriteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := URLs
	URLs = []string{srv.URL + "/ips-v4"}
	defer func() { URLs = old }()

	var buf bytes.Buffer
	if err := Write(&buf, srv.Client()); err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}
