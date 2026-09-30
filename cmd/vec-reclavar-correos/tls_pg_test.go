package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Negocia realmente SSLRequest PostgreSQL y TLS. No consulta una base ni
// abre servicios ajenos: el servidor sintético vive en loopback efímero.
func TestConexionPostgreSQLTLSRealYNombreAjenoDenegado(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	h := sha256.Sum256(ca)
	for _, ajeno := range []bool{false, true} {
		t.Run(fmt.Sprint(ajeno), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				var request [8]byte
				if _, err := io.ReadFull(conn, request[:]); err != nil || binary.BigEndian.Uint32(request[4:]) != 80877103 {
					return
				}
				if _, err := conn.Write([]byte("S")); err != nil {
					return
				}
				s := tls.Server(conn, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}})
				if s.Handshake() != nil {
					return
				}
				var length [4]byte
				if _, err := io.ReadFull(s, length[:]); err != nil {
					return
				}
				n := binary.BigEndian.Uint32(length[:])
				if n < 8 || n > 16384 {
					return
				}
				if _, err := io.CopyN(io.Discard, s, int64(n-4)); err != nil {
					return
				}
				// Trust sintético, parámetros y ReadyForQuery.
				_, _ = s.Write([]byte{'R', 0, 0, 0, 8, 0, 0, 0, 0, 'Z', 0, 0, 0, 5, 'I'})
				_, _ = io.Copy(io.Discard, s)
			}()
			p := planPrueba()
			p.Conexion.Puerto = uint16(listener.Addr().(*net.TCPAddr).Port)
			p.Conexion.SSLMode = "verify-full"
			p.Conexion.CAHuella = hex.EncodeToString(h[:])
			p.ConexionHuella = huellaConexion(p.Base, p.Conexion) // #nosec G115 -- puerto TCP asignado por SO 1..65535.
			dsn := []byte(fmt.Sprintf("postgresql://operador@127.0.0.1:%d/sintetica?sslmode=verify-full", p.Conexion.Puerto))
			cfg, err := configuracionConexion(dsn, p, ca)
			if err != nil {
				t.Fatal(err)
			}
			if ajeno {
				cfg.TLSConfig.ServerName = "otro.example.invalid"
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			conn, err := pgx.ConnectConfig(ctx, cfg)
			if ajeno && err == nil {
				conn.Close(ctx)
				t.Fatal("nombre ajeno aceptado")
			}
			if !ajeno {
				if err != nil {
					t.Fatal(err)
				}
				conn.Close(ctx)
			}
			listener.Close()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("servidor sintético no terminó")
			}
		})
	}
}
