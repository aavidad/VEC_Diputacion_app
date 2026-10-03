package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestTLSVerificaNombreYCAAprobados(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	h := sha256.Sum256(ca)
	p := planPrueba()
	p.Conexion = destinoConexion{Host: "clon.example.invalid", Puerto: 5432, Usuario: "operador", SSLMode: "verify-full", CAHuella: hex.EncodeToString(h[:])}
	p.ConexionHuella = huellaConexion(p.Base, p.Conexion)
	c, err := configuracionConexion([]byte("postgresql://operador@clon.example.invalid:5432/sintetica?sslmode=verify-full"), p, ca)
	if err != nil || c.TLSConfig == nil || c.TLSConfig.InsecureSkipVerify || c.TLSConfig.ServerName != "clon.example.invalid" || c.TLSConfig.MinVersion < tls.VersionTLS12 || len(c.Fallbacks) != 0 {
		t.Fatal("TLS no verificable", err)
	}
	if _, err := configuracionConexion([]byte("postgresql://operador@clon.example.invalid:5432/sintetica?sslmode=verify-full"), p, []byte("CA distinta")); err == nil {
		t.Fatal("CA ajena aceptada")
	}
	p.Conexion.SSLMode = "disable"
	p.Conexion.ClonLocal = true
	p.Conexion.CAHuella = ""
	p.ConexionHuella = huellaConexion(p.Base, p.Conexion)
	if _, err := configuracionConexion([]byte("postgresql://operador@clon.example.invalid:5432/sintetica?sslmode=disable"), p, nil); err == nil {
		t.Fatal("remoto sin TLS aceptado")
	}
}
