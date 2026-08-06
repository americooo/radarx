package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// CommonPorts is a compact set of ports worth checking on a recon sweep —
// web, mail, remote access, databases, and common dev/admin services.
var CommonPorts = []int{
	21, 22, 25, 53, 80, 110, 143, 443, 445, 465, 587, 993, 995,
	1433, 1521, 2049, 2375, 3000, 3306, 3389, 5432, 5601, 5672,
	5900, 6379, 8000, 8008, 8080, 8443, 8888, 9000, 9092, 9200, 11211, 27017,
}

// ScanPorts runs a TCP connect scan against a host for the given ports.
// A successful TCP handshake means the port is open. This is a light,
// non-intrusive check (no raw packets, no SYN scan) so it stays within the
// bounds of ordinary authorized recon.
func ScanPorts(ctx context.Context, host string, ports []int, workers int) []model.Asset {
	if len(ports) == 0 {
		ports = CommonPorts
	}
	if workers <= 0 {
		workers = 50
	}

	jobs := make(chan int)
	out := make(chan int)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := net.Dialer{Timeout: 3 * time.Second}
			for port := range jobs {
				addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
				conn, err := d.DialContext(ctx, "tcp", addr)
				if err == nil {
					conn.Close()
					out <- port
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range ports {
			select {
			case <-ctx.Done():
				return
			case jobs <- p:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()

	var assets []model.Asset
	for port := range out {
		assets = append(assets, model.Asset{
			Kind: model.KindPort,
			Key:  fmt.Sprintf("%s:%d", host, port),
			Host: host,
			Port: port,
		})
	}
	return assets
}
