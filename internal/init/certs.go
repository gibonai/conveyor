package init

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// generateCertificates creates a CA certificate and a server certificate signed by the CA
func generateCertificates(certDir string, force bool) error {
	caKeyPath := filepath.Join(certDir, "ca-key.pem")
	caCertPath := filepath.Join(certDir, "ca.crt")
	serverKeyPath := filepath.Join(certDir, "server-key.pem")
	serverCertPath := filepath.Join(certDir, "server.crt")

	// Check if certificates already exist
	if !force {
		if fileExists(caCertPath) || fileExists(serverCertPath) {
			return fmt.Errorf("certificates already exist (use --force to overwrite)")
		}
	}

	fmt.Println("Generating CA certificate...")
	
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:    "Conveyor CI Root CA",
			Organization:  []string{"Conveyor CI"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Write CA certificate
	if err := writeCertificate(caCertPath, caCertDER); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Write CA private key
	if err := writePrivateKey(caKeyPath, caKey); err != nil {
		return fmt.Errorf("failed to write CA private key: %w", err)
	}

	fmt.Println("Generating server certificate...")

	// Generate server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Create server certificate template
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   "Conveyor CI Server",
			Organization: []string{"Conveyor CI"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:     []string{"localhost", "conveyor", "conveyor.local"},
	}

	// Parse CA certificate for signing
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Create server certificate
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Write server certificate
	if err := writeCertificate(serverCertPath, serverCertDER); err != nil {
		return fmt.Errorf("failed to write server certificate: %w", err)
	}

	// Write server private key
	if err := writePrivateKey(serverKeyPath, serverKey); err != nil {
		return fmt.Errorf("failed to write server private key: %w", err)
	}

	return nil
}

// copyExistingCertificates copies existing certificates to the cert directory
func copyExistingCertificates(caPath, keyPath, certPath, certDir string) error {
	// Verify files exist
	if !fileExists(caPath) {
		return fmt.Errorf("CA certificate file does not exist: %s", caPath)
	}
	if !fileExists(keyPath) {
		return fmt.Errorf("private key file does not exist: %s", keyPath)
	}
	if !fileExists(certPath) {
		return fmt.Errorf("server certificate file does not exist: %s", certPath)
	}

	// Copy files
	if err := copyFile(caPath, filepath.Join(certDir, "ca.crt")); err != nil {
		return fmt.Errorf("failed to copy CA certificate: %w", err)
	}
	if err := copyFile(keyPath, filepath.Join(certDir, "server-key.pem")); err != nil {
		return fmt.Errorf("failed to copy server private key: %w", err)
	}
	if err := copyFile(certPath, filepath.Join(certDir, "server.crt")); err != nil {
		return fmt.Errorf("failed to copy server certificate: %w", err)
	}

	return nil
}

// writeCertificate writes a certificate to file in PEM format
func writeCertificate(filename string, certDER []byte) error {
	certFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create certificate file %s: %w", filename, err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}); err != nil {
		return fmt.Errorf("failed to encode certificate: %w", err)
	}

	return nil
}

// writePrivateKey writes a private key to file in PEM format
func writePrivateKey(filename string, key *rsa.PrivateKey) error {
	keyFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file %s: %w", filename, err)
	}
	defer keyFile.Close()

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	}); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get file info to preserve permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}