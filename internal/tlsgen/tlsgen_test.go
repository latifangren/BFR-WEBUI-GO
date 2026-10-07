package tlsgen

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCertificates(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := EnsureCertificates(tempDir)
	if err != nil {
		t.Fatalf("EnsureCertificates failed: %v", err)
	}

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("failed to read generated cert: %v", err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read generated key: %v", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("expected valid CERTIFICATE PEM block, got %v", block)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse x509 certificate: %v", err)
	}

	if cert.Subject.CommonName != "bfr.local" {
		t.Errorf("expected CommonName bfr.local, got %s", cert.Subject.CommonName)
	}

	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" {
		t.Fatalf("expected valid EC PRIVATE KEY PEM block, got %v", keyBlock)
	}

	// Calling again should detect existing certs and return existing paths with nil error
	c2, k2, err := EnsureCertificates(tempDir)
	if err != nil {
		t.Fatalf("EnsureCertificates on existing certs failed: %v", err)
	}
	if c2 != certPath || k2 != keyPath {
		t.Errorf("expected same paths returned, got (%s, %s)", c2, k2)
	}
}

func TestEnsureCertificatesDefaultDirAndIPs(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_MODULE_DIR", tempDir)

	c, k, err := EnsureCertificates("")
	if err != nil {
		t.Fatalf("EnsureCertificates with empty certDir failed: %v", err)
	}
	if c == "" || k == "" {
		t.Errorf("expected non-empty cert and key paths, got (%s, %s)", c, k)
	}

	ips, _ := getInterfaceIPs()
	t.Logf("Found %d interface IPs", len(ips))
}

func TestEnsureCertificatesFallbackDir(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "dummy_file")
	_ = os.WriteFile(filePath, []byte("data"), 0644)

	// Passing a path where a file already exists causes MkdirAll to fail and trigger fallback
	c, k, err := EnsureCertificates(filepath.Join(filePath, "impossible_dir"))
	if err == nil {
		defer func() {
			_ = os.Remove(c)
			_ = os.Remove(k)
			_ = os.Remove("./tls")
		}()
		t.Logf("Fallback succeeded: %s, %s", c, k)
	}
}

func TestEnsureCertificatesPartialFiles(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "cert.pem")
	_ = os.WriteFile(certPath, []byte("only cert exists"), 0644)

	// One file exists, should regenerate properly
	c, k, err := EnsureCertificates(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == "" || k == "" {
		t.Errorf("expected paths, got (%s, %s)", c, k)
	}
}
