package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBancoNoSaleDelOrigenNiEjecutaActos(t *testing.T) {
	for _, base := range []string{"http://127.0.0.1", "https://example.org", "https://usuario@localhost", "https://localhost/ruta", "https://localhost?otra=1"} {
		if validarDestino(base) == nil {
			t.Fatalf("aceptó %s", base)
		}
	}
	for _, base := range []string{"https://localhost:18480", "https://127.0.0.1:18480", "https://[::1]:18480"} {
		if err := validarDestino(base); err != nil {
			t.Fatal(err)
		}
	}
	for _, caso := range []casoLectura{
		{Ruta: "//example.org", Metodo: "GET"},
		{Ruta: "https://example.org", Metodo: "GET"},
		{Ruta: "/api/vec/contratacion-temporal/altas", Metodo: "POST", Cuerpo: json.RawMessage(`{}`)},
		{Ruta: "/api/vec/contratacion-temporal/cuadro/consultas", Metodo: "DELETE"},
		{Ruta: "/api/vec/contratacion-temporal/cuadro/consultas?x=1", Metodo: "POST", Cuerpo: json.RawMessage(`{}`)},
	} {
		if validarCaso(&caso) == nil {
			t.Fatalf("aceptó %s %s", caso.Metodo, caso.Ruta)
		}
	}
	dir := t.TempDir()
	fichero := filepath.Join(dir, "casos.json")
	for _, contenido := range []string{`[]`, `[ {"ruta":"/a","nombre":"igual"}, {"ruta":"/b","nombre":"igual"} ]`, `[{"ruta":"/a","cabeceras":{"Authorization":"secreto"}}]`, `[{"ruta":"/a"}] {}`} {
		if err := os.WriteFile(fichero, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := cargarCasos(fichero, nil); err == nil {
			t.Fatal("aceptó manifiesto inválido")
		}
	}
}

func TestPOSTConMTLSYRespuestaCompleta(t *testing.T) {
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "cliente de prueba"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	certificado, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	raicesCliente := x509.NewCertPool()
	raicesCliente.AddCert(certificado)
	servidor := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" || len(r.TLS.PeerCertificates) != 1 {
			t.Error("no llegó la consulta nominal")
		}
		var cuerpo map[string]int
		if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil || cuerpo["limite"] != 100 {
			t.Error("cuerpo alterado")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"data":{}}`)); err != nil {
			t.Error(err)
		}
	}))
	servidor.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raicesCliente, MinVersion: tls.VersionTLS12}
	servidor.StartTLS()
	defer servidor.Close()
	raicesServidor := x509.NewCertPool()
	raicesServidor.AddCert(servidor.Certificate())
	conf := &tls.Config{RootCAs: raicesServidor, MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: clave}}}
	cliente, tr := nuevoCliente(conf)
	defer tr.CloseIdleConnections()
	caso := casoLectura{Metodo: "POST", Ruta: "/api/vec/contratacion-temporal/cuadro/consultas", Cuerpo: json.RawMessage(`{"limite":100}`)}
	if _, fallo := medir(cliente, servidor.URL, caso); fallo != "" {
		t.Fatal(fallo)
	}
	sinCert, trSin := nuevoCliente(&tls.Config{RootCAs: raicesServidor, MinVersion: tls.VersionTLS12})
	defer trSin.CloseIdleConnections()
	if _, fallo := medir(sinCert, servidor.URL, caso); fallo != "red" {
		t.Fatalf("sin certificado esperado=red obtenido=%s", fallo)
	}
}

func TestErroresYRedireccionNoSeCuentanComoExito(t *testing.T) {
	visitas := 0
	externo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { visitas++ }))
	defer externo.Close()
	servidor := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redireccion":
			http.Redirect(w, r, externo.URL, http.StatusFound)
		case "/truncado":
			w.Header().Set("Content-Length", "100")
			if _, err := w.Write([]byte("corto")); err != nil {
				t.Error(err)
			}
		default:
			w.WriteHeader(503)
		}
	}))
	defer servidor.Close()
	raices := x509.NewCertPool()
	raices.AddCert(servidor.Certificate())
	cliente, tr := nuevoCliente(&tls.Config{RootCAs: raices, MinVersion: tls.VersionTLS12})
	defer tr.CloseIdleConnections()
	for ruta, esperado := range map[string]string{"/redireccion": "302", "/truncado": "cuerpo", "/fallo": "503"} {
		if _, fallo := medir(cliente, servidor.URL, casoLectura{Metodo: "GET", Ruta: ruta}); fallo != esperado {
			t.Fatalf("%s esperado=%s obtenido=%s", ruta, esperado, fallo)
		}
	}
	if visitas != 0 {
		t.Fatal("siguió redirección fuera del origen")
	}
	casos := []casoLectura{{Nombre: "fallo", Metodo: "GET", Ruta: "/fallo", PresupuestoMS: 300}}
	if tanda(servidor.URL, &tls.Config{RootCAs: raices, MinVersion: tls.VersionTLS12}, 1, time.Second, 0, 2, 1, casos) {
		t.Fatal("banco dio éxito con 503")
	}
}

func TestPercentilNearestRank(t *testing.T) {
	valores := make([]float64, 30)
	for i := range valores {
		valores[i] = float64(i + 1)
	}
	if percentil(valores, .95) != 29 || percentil(valores, .5) != 15 || !math.IsNaN(percentil(nil, .95)) {
		t.Fatal("percentil incorrecto")
	}
	casos, err := cargarCasos("", []string{"/api/vec/session"})
	if err != nil || len(casos) != 1 || !strings.HasPrefix(casos[0].Nombre, "/api") {
		t.Fatal("compatibilidad GET")
	}
}
