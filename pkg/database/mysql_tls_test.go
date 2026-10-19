package database

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mysqlTestCertificate(t *testing.T, expired, wrongName bool) (tls.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test MySQL CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	root, err := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, KeyUsage: x509.KeyUsageDigitalSignature, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	if expired {
		leaf.NotAfter = now.Add(-time.Minute)
	}
	if wrongName {
		leaf.IPAddresses = []net.IP{net.ParseIP("127.0.0.2")}
	}
	der, err := x509.CreateCertificate(rand.Reader, leaf, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root}), 0o600); err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der, root}, PrivateKey: key}, path
}

// A protocol peer verifies the driver's actual TLS negotiation, not just flags.
// It never implements authentication or runs a SQL statement.
func TestMySQLConnectorRequiresVerifiedTLSOnEveryConnection(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		expired, wrongName, noTLS bool
	}{
		{name: "valid"}, {name: "expired", expired: true}, {name: "wrong hostname", wrongName: true}, {name: "server has no TLS", noTLS: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, ca := mysqlTestCertificate(t, tc.expired, tc.wrongName)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			connector, err := NewMySQLConnector(&MySQLConfig{Host: listener.Addr().String(), Username: "test", Password: "never sent before TLS", Database: "db", TLSMode: "VERIFY_IDENTITY", TLSCAFile: ca})
			if err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				verified := make(chan bool, 1)
				go func() {
					conn, err := listener.Accept()
					if err != nil {
						verified <- false
						return
					}
					defer conn.Close()
					_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
					caps := uint32(0x00000200 | 0x00008000 | 0x00080000)
					if !tc.noTLS {
						caps |= 0x00000800
					}
					greeting := append([]byte{10}, []byte("8.0.36-test\x00")...)
					greeting = append(greeting, 1, 0, 0, 0)
					greeting = append(greeting, []byte("12345678\x00")...)
					greeting = append(greeting, byte(caps), byte(caps>>8), 45, 2, 0, byte(caps>>16), byte(caps>>24), 21)
					greeting = append(greeting, make([]byte, 10)...)
					greeting = append(greeting, []byte("123456789012\x00mysql_native_password\x00")...)
					_, _ = conn.Write(append([]byte{byte(len(greeting)), 0, 0, 0}, greeting...))
					header := make([]byte, 4)
					if _, err := io.ReadFull(conn, header); err != nil {
						verified <- false
						return
					}
					n := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
					request := make([]byte, n)
					if _, err := io.ReadFull(conn, request); err != nil {
						verified <- false
						return
					}
					if tc.noTLS || n != 32 || binary.LittleEndian.Uint32(request)&0x800 == 0 {
						verified <- false
						return
					}
					secure := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
					verified <- secure.Handshake() == nil
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				connection, _ := connector.Connect(ctx)
				cancel()
				if connection != nil {
					_ = connection.Close()
				}
				want := !tc.expired && !tc.wrongName && !tc.noTLS
				if got := <-verified; got != want {
					t.Fatalf("attempt %d: TLS verified=%v, want %v", attempt, got, want)
				}
			}
		})
	}
}

func TestMySQLTLSConfigurationRejectsInvalidInputs(t *testing.T) {
	for _, cfg := range []*MySQLConfig{
		{Host: "db:3306", TLSMode: "PREFERRED"},
		{Host: "db:3306", TLSMode: "VERIFY_IDENTITY"},
		{Host: "db:3306", TLSMode: "VERIFY_IDENTITY", TLSCAFile: "/no/such/mysql-ca.pem"},
	} {
		if _, err := NewMySQLConnector(cfg); err == nil {
			t.Fatal("invalid TLS configuration accepted")
		}
	}
	_, ca := mysqlTestCertificate(t, false, false)
	if err := os.WriteFile(ca, []byte("not a CA"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMySQLConnector(&MySQLConfig{Host: "db:3306", TLSMode: "VERIFY_IDENTITY", TLSCAFile: ca}); err == nil {
		t.Fatal("invalid CA accepted")
	}
}
