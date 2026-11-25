// Package ipfetch downloads Cloudflare's published public IP ranges.
package ipfetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// URLs are the authoritative, well-known endpoints where Cloudflare publishes
// its public anycast ranges (IPv4 first, then IPv6).
var URLs = []string{
	"https://www.cloudflare.com/ips-v4",
	"https://www.cloudflare.com/ips-v6",
}

// Write downloads every range list in URLs and writes them, one CIDR per line,
// in order (IPv4 first, then IPv6). The client and URLs are injectable for
// testing.
func Write(w io.Writer, client *http.Client) error {
	for _, url := range URLs {
		resp, err := client.Get(url)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", url, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read %s: %w", url, readErr)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
		}
		if _, err := w.Write(body); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		if len(body) == 0 || body[len(body)-1] != '\n' {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}
	return nil
}

// ToFile downloads Cloudflare's public IP ranges into path.
func ToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	if err := Write(f, client); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
