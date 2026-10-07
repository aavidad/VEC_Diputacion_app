package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
	ports "vec-diputacion-granada/internal/modules/personal/ports"
)

// This fixture exercises the actual CLI factory and mTLS transport. It supplies
// an ephemeral API response, not a real F1/PDP/PostgreSQL installation.
func TestConsultaNominalFabricaRealGETMTLS(t *testing.T) {
	dir, err := os.MkdirTemp("/var/tmp", "oh-cli-tls-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	}()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	pair := func(serial int64, server bool) (tls.Certificate, []byte, []byte) {
		k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		leaf := &x509.Certificate{SerialNumber: big.NewInt(serial), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
		if server {
			leaf.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
			leaf.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		} else {
			leaf.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
		d, e := x509.CreateCertificate(rand.Reader, leaf, ca, &k.PublicKey, caKey)
		if e != nil {
			t.Fatal(e)
		}
		pk, e := x509.MarshalPKCS8PrivateKey(k)
		if e != nil {
			t.Fatal(e)
		}
		c, p := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: d}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pk})
		par, e := tls.X509KeyPair(c, p)
		if e != nil {
			t.Fatal(e)
		}
		return par, c, p
	}
	serverPair, _, _ := pair(2, true)
	_, cp, kp := pair(3, false)
	for name, b := range map[string][]byte{"ca.pem": caPEM, "client.pem": cp, "client.key": kp} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA")
	}
	var requests atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := requests.Add(1)
		if r.Method != "GET" || r.URL.Path != "/api/vec/personal/organizacion-historica" || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || r.URL.Query().Get("organismo_ref") != "" || r.Header.Get("Authorization") != "" {
			t.Error("frontera alterada")
			w.WriteHeader(500)
			return
		}
		q := r.URL.Query()
		conocido, e := time.Parse("2006-01-02T15:04:05.000000Z", q.Get("conocido_en"))
		if e != nil {
			t.Error(e)
			w.WriteHeader(400)
			return
		}
		s := personal.SelectorOrganizacionHistorica{OrganismoRef: "org_prueba", UnidadClave: q.Get("unidad_clave"), VigenteEn: personal.FechaCivil(q.Get("vigente_en")), ConocidoEn: conocido, Limite: 2}
		cov := ports.CoberturaFuentesOrganizacionHistorica{Unidades: "sin_datos", PuestosTipo: "sin_datos", Dotaciones: "sin_datos", Plazas: "sin_datos", PuestosIndividuales: "sin_datos", Vinculos: "sin_datos"}
		id := strconv.Itoa(int(seq))
		result := ports.ResultadoConsultaOrganizacionHistorica{Pagina: ports.PaginaOrganizacionHistorica{Selector: s, Cobertura: cov}, Evidencia: ports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: "recibo:" + id, DecisionRef: "decision:" + id, EfectoRef: s.OrganismoRef, AuditoriaRef: "auditoria:" + id, ConsumoHuellaSHA256: strings.Repeat(id, 64), ConsultadaEn: now.UTC().Truncate(time.Microsecond)}}
		w.Header().Set("Content-Type", "application/json")
		if e := json.NewEncoder(w).Encode(map[string]any{"data": result}); e != nil {
			t.Error(e)
		}
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{serverPair}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	server.StartTLS()
	defer server.Close()
	cfg := configuracionConsultaNominal{Version: 1, Origen: server.URL, AutoridadCA: "ca.pem", CertificadoCliente: "client.pem", ClaveCliente: "client.key", MaximoBytesPagina: 1 << 20}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "consulta.json")
	if err := os.WriteFile(path, cfgJSON, 0600); err != nil {
		t.Fatal(err)
	}
	a := personal.SelectorOrganizacionHistorica{OrganismoRef: "org_prueba", UnidadClave: "unidad_prueba", VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), Limite: 2}
	d := a
	d.VigenteEn = "2026-10-02"
	input, err := json.Marshal(entradaConsultaNominal{Esquema: esquemaConsultaNominal, Formato: "json", Antes: a, Despues: d})
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if status := ejecutar([]string{"-consulta-nominal", path}, bytes.NewReader(input), &out, &stderr); status != 0 || requests.Load() != 2 || stderr.Len() != 0 {
		t.Fatalf("status%d llamadas%d error%s", status, requests.Load(), stderr.String())
	}
	var r salidaConsultaNominal
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Manifiesto.Modo != "consulta_autorizada" || r.Manifiesto.PaginasAntes != 1 || r.Manifiesto.PaginasDespues != 1 || r.Manifiesto.TotalHechosAntes != 0 || strings.Contains(out.String(), dir) || strings.Contains(out.String(), server.URL) || strings.Contains(out.String(), "sintetico") {
		t.Fatal("manifiesto nominal no minimizado")
	}
}
