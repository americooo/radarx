package engine

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// InspectCert connects to host:443 and reads the leaf TLS certificate without
// trusting it. Useful for spotting new certs (a strong signal a new service
// went live) and expiry tracking. Returns (asset, true) if a cert was read.
func InspectCert(ctx context.Context, host string) (model.Asset, bool) {
	d := &net.Dialer{Timeout: 6 * time.Second}
	rawConn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, "443"))
	if err != nil {
		return model.Asset{}, false
	}
	defer rawConn.Close()

	conn := tls.Client(rawConn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, //nolint:gosec // recon: we inspect, we don't trust
	})
	_ = conn.SetDeadline(time.Now().Add(6 * time.Second))
	if err := conn.HandshakeContext(ctx); err != nil {
		return model.Asset{}, false
	}
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return model.Asset{}, false
	}
	leaf := state.PeerCertificates[0]

	return model.Asset{
		Kind:       model.KindCert,
		Key:        host + "|cert",
		Host:       host,
		CertCN:     leaf.Subject.CommonName,
		CertExpiry: leaf.NotAfter.UTC().Format(time.RFC3339),
	}, true
}
