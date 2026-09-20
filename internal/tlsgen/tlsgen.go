// Package tlsgen provides self-signed TLS certificate generation for BFR WEBUI GO.
package tlsgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"bfr-webui-go/internal/config"
)

// EnsureCertificates checks if TLS cert and key exist at certDir.
// If missing, it generates a self-signed ECDSA P-256 certificate with local IP/DNS SANs.
func EnsureCertificates(certDir string) (certPath string, keyPath string, err error) {
	if certDir == "" {
		certDir = filepath.Join(config.ModuleDir, "tls")
	}

	certPath = filepath.Join(certDir, "cert.pem")
	keyPath = filepath.Join(certDir, "key.pem")

	if fileExistsAndNotEmpty(certPath) && fileExistsAndNotEmpty(keyPath) {
		return certPath, keyPath, nil
	}

	if err := os.MkdirAll(certDir, 0755); err != nil {
		// Fallback to local ./tls directory
		certDir = "./tls"
		certPath = filepath.Join(certDir, "cert.pem")
		keyPath = filepath.Join(certDir, "key.pem")
		if err := os.MkdirAll(certDir, 0755); err != nil {
			return "", "", fmt.Errorf("tlsgen: failed to create cert directory: %w", err)
		}
	}

	// Generate ECDSA P-256 private key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to generate private key: %w", err)
	}

	// Serial number
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNum, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to generate serial number: %w", err)
	}

	// Collect local network interface IPs for Subject Alternative Names (SANs)
	ipSANs := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("192.168.43.1"), // Default Android hotspot
		net.ParseIP("192.168.44.1"),
		net.ParseIP("::1"),
	}
	if ifaceIPs, err := getInterfaceIPs(); err == nil {
		ipSANs = append(ipSANs, ifaceIPs...)
	}

	template := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"BFR WEBUI GO"},
			CommonName:   "bfr.local",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Valid for 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "bfr.local", "pixel.local", "android.local"},
		IPAddresses:           ipSANs,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to create certificate: %w", err)
	}

	// Write Certificate PEM
	certOut, err := os.OpenFile(certPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to open cert file for writing: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to write cert pem: %w", err)
	}

	// Write Private Key PEM
	keyOut, err := os.OpenFile(keyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to open key file for writing: %w", err)
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return "", "", fmt.Errorf("tlsgen: failed to write key pem: %w", err)
	}

	return certPath, keyPath, nil
}

func fileExistsAndNotEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

func getInterfaceIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	seen := make(map[string]bool)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ipStr := ip.String()
			if !seen[ipStr] {
				seen[ipStr] = true
				ips = append(ips, ip)
			}
		}
	}
	return ips, nil
}
