package bootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAdscripcionCentroDesarrolloAusente(t *testing.T) {
	if got, err := cargarAdscripcionesCentrosDesarrollo(t.TempDir(), nil); err != nil || got.Adscripciones != nil {
		t.Fatalf("ausente: got=%v err=%v", got, err)
	}
}

func TestResolvedorAdscripcionCentroCopiaPorValor(t *testing.T) {
	r := &resolvedorIdentidadDesarrollo{porSujeto: map[string]adscripcionCentroDesarrollo{
		"sol": {CentroRef: "centro-a", PuestoRef: "puesto-a", RatificadorSubject: "rat"},
	}}
	got, ok := r.adscripcionCentro("sol")
	if !ok || got.CentroRef != "centro-a" || got.PuestoRef != "puesto-a" || got.RatificadorSubject != "rat" {
		t.Fatalf("adscripcion=%+v ok=%v", got, ok)
	}
	if _, ok := r.adscripcionCentro("desconocido"); ok {
		t.Fatal("sujeto desconocido aceptado")
	}
}

func TestAdscripcionCentroCertificadosYPares(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	escribir := func(ruta string, v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ruta), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	archivo := archivoCentrosDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa}
	for i, rol := range []string{"solicitante_centro", "ratificador_centro"} {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		leaf := &x509.Certificate{SerialNumber: big.NewInt(int64(i + 2)), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
		der, err := x509.CreateCertificate(rand.Reader, leaf, ca, pub, priv)
		if err != nil {
			t.Fatal(err)
		}
		certPath := "identidad/" + rol + ".crt"
		idPath := "identidad/" + rol + ".json"
		if err := os.WriteFile(filepath.Join(root, certPath), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(der)
		escribir(idPath, archivoIdentidadDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Subject: rol, DisplayName: "Persona sintética " + rol, Roles: []string{rol}, CertificateSHA256: hex.EncodeToString(h[:])})
		e := entradaCentroDesarrollo{Certificate: certPath, Identity: idPath, Role: rol, CentroRef: "centro-520", PuestoRef: "puesto:prueba:" + rol}
		if i == 0 {
			e.RatificadorSubject = "ratificador_centro"
		}
		archivo.Entradas = append(archivo.Entradas, e)
	}
	escribir("identidad/centros.json", archivo)
	got, err := cargarAdscripcionesCentrosDesarrollo(root, ca)
	if err != nil || len(got.Identidades) != 2 || len(got.Adscripciones) != 2 {
		t.Fatalf("par válido: %v", err)
	}
	if _, err := nuevoResolvedorIdentidadDesarrollo(got.Identidades...); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarAdscripcionesCentrosDesarrollo(root, ca, got.Identidades[0]); err == nil {
		t.Fatal("acepta sujeto existente")
	}
	for nombre, cambio := range map[string]func(*archivoCentrosDesarrollo){
		"centro distinto":     func(a *archivoCentrosDesarrollo) { a.Entradas[1].CentroRef = "centro-otro" },
		"autorratificación":   func(a *archivoCentrosDesarrollo) { a.Entradas[0].RatificadorSubject = "solicitante_centro" },
		"sujeto repetido":     func(a *archivoCentrosDesarrollo) { a.Entradas[1] = a.Entradas[0] },
		"sin ratificador":     func(a *archivoCentrosDesarrollo) { a.Entradas = a.Entradas[:1] },
		"rol ajeno":           func(a *archivoCentrosDesarrollo) { a.Entradas[1].Role = "tecnico_rrhh" },
		"ruta ascendente":     func(a *archivoCentrosDesarrollo) { a.Entradas[1].Identity = ".." },
		"referencia inválida": func(a *archivoCentrosDesarrollo) { a.Entradas[0].CentroRef = "centro\t520" },
	} {
		t.Run(nombre, func(t *testing.T) {
			a := archivo
			a.Entradas = append([]entradaCentroDesarrollo(nil), archivo.Entradas...)
			cambio(&a)
			escribir("identidad/centros.json", a)
			if _, err := cargarAdscripcionesCentrosDesarrollo(root, ca); err == nil {
				t.Fatal("configuración inválida admitida")
			}
		})
	}
	if err := os.Remove(filepath.Join(root, "identidad/centros.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "ausente"), filepath.Join(root, "identidad/centros.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarAdscripcionesCentrosDesarrollo(root, ca); err == nil {
		t.Fatal("enlace colgante interpretado como configuración ausente")
	}
}
