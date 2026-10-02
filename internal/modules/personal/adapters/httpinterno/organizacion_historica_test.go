package httpinterno

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

func selectorPruebaOH() domain.SelectorOrganizacionHistorica {
	return domain.SelectorOrganizacionHistorica{OrganismoRef: "org_prueba", UnidadClave: "unidad_prueba", VigenteEn: "2026-10-02", ConocidoEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), Limite: 2}
}

func paginaPruebaOH(s domain.SelectorOrganizacionHistorica) ports.ResultadoConsultaOrganizacionHistorica {
	cov := ports.CoberturaFuentesOrganizacionHistorica{Unidades: "sin_datos", PuestosTipo: "sin_datos", Dotaciones: "sin_datos", Plazas: "sin_datos", PuestosIndividuales: "sin_datos", Vinculos: "sin_datos"}
	return ports.ResultadoConsultaOrganizacionHistorica{Pagina: ports.PaginaOrganizacionHistorica{Selector: s, Cobertura: cov}, Evidencia: ports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: "recibo:prueba", DecisionRef: "decision:prueba", EfectoRef: s.OrganismoRef, AuditoriaRef: "auditoria:prueba", ConsumoHuellaSHA256: strings.Repeat("a", 64), ConsultadaEn: s.ConocidoEn}}
}

func bodyPruebaOH(t *testing.T, r ports.ResultadoConsultaOrganizacionHistorica) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"data": r})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The certificates are generated exclusively for the loopback fixture. The
// client uses normal chain verification and the server requires client auth.
func fixtureTLSOH(t *testing.T, h http.Handler) ConfiguracionOrganizacionHistorica {
	t.Helper()
	dir, err := os.MkdirTemp("/var/tmp", "oh-tls-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certCA, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	makePair := func(serial int64, server bool) (tls.Certificate, []byte, []byte) {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		c := &x509.Certificate{SerialNumber: big.NewInt(serial), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
		if server {
			c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			c.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		} else {
			c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
		d, err := x509.CreateCertificate(rand.Reader, c, certCA, &k.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		pk, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			t.Fatal(err)
		}
		cp, kp := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: d}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pk})
		p, err := tls.X509KeyPair(cp, kp)
		if err != nil {
			t.Fatal(err)
		}
		return p, cp, kp
	}
	serverPair, _, _ := makePair(2, true)
	_, clientPEM, keyPEM := makePair(3, false)
	for n, b := range map[string][]byte{"ca.pem": caPEM, "client.pem": clientPEM, "client.key": keyPEM} {
		if err := os.WriteFile(filepath.Join(dir, n), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA")
	}
	server := httptest.NewUnstartedServer(h)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{serverPair}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	server.StartTLS()
	t.Cleanup(server.Close)
	return ConfiguracionOrganizacionHistorica{Origen: server.URL, Directorio: dir, AutoridadCA: "ca.pem", CertificadoCliente: "client.pem", ClaveCliente: "client.key", MaximoBytesPagina: 1 << 20}
}

func TestClienteOHMTLSYSelectorSinAutoridadDelCliente(t *testing.T) {
	s := selectorPruebaOH()
	var llamadas atomic.Int32
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	cfg := fixtureTLSOH(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		if r.Method != "GET" || r.URL.Path != rutaOrganizacionHistorica || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) != 1 || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Referer") != "" || r.URL.Query().Get("organismo_ref") != "" || r.URL.Query().Get("actor") != "" || r.URL.Query().Get("perfil") != "" || r.URL.Query().Get("conocido_en") != "2026-10-02T12:00:00.000000Z" || r.URL.Query().Get("unidad_clave") != s.UnidadClave {
			t.Error("petición nominal alterada")
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := w.Write(bodyPruebaOH(t, paginaPruebaOH(s))); err != nil {
			t.Error(err)
		}
	}))
	c, err := NuevoClienteOrganizacionHistorica(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Cerrar()
	r, err := c.Consultar(context.Background(), s)
	if err != nil || r.Evidencia.EfectoRef != s.OrganismoRef || r.Pagina.Cobertura.Unidades != "sin_datos" || llamadas.Load() != 1 {
		t.Fatalf("resultado=%v err=%v llamadas=%d", r, err, llamadas.Load())
	}
}

func TestClienteOHRechazaOrigenYLimitesAntesDeLeerTLS(t *testing.T) {
	for _, origen := range []string{"http://localhost", "https://operador@localhost", "https://localhost/", "https://LOCALHOST", "https://localhost?x=1", "https://localhost#x", "https://localhost:", "https://localhost:01", "https://localhost:65536"} {
		if _, err := NuevoClienteOrganizacionHistorica(ConfiguracionOrganizacionHistorica{Origen: origen, MaximoBytesPagina: 1}); err == nil {
			t.Fatalf("origen aceptado %s", origen)
		}
	}
	for _, max := range []int64{-1, 0, math.MaxInt64} {
		if _, err := NuevoClienteOrganizacionHistorica(ConfiguracionOrganizacionHistorica{Origen: "https://localhost", MaximoBytesPagina: max}); err == nil {
			t.Fatal("límite sin cota u overflow aceptado")
		}
	}
}

func TestClienteOHRechazaRevocacionYRespuestaNoNominal(t *testing.T) {
	s := selectorPruebaOH()
	for _, status := range []int{401, 403, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cfg := fixtureTLSOH(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				if _, err := w.Write([]byte("ruta y datos privados")); err != nil {
					t.Error(err)
				}
			}))
			c, err := NuevoClienteOrganizacionHistorica(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Cerrar()
			r, err := c.Consultar(context.Background(), s)
			if err == nil || r.Evidencia.ReciboRef != "" || strings.Contains(err.Error(), "privados") || strings.Contains(err.Error(), cfg.Origen) {
				t.Fatal("fallo con salida parcial o metadatos")
			}
			if status != 503 && !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaDenegada) {
				t.Fatal("denegación no conservada")
			}
		})
	}
}

func TestClienteOHRechazaManipulacionYExceso(t *testing.T) {
	s := selectorPruebaOH()
	for _, tc := range []struct {
		nombre    string
		mutar     func(*ports.ResultadoConsultaOrganizacionHistorica)
		raw       func([]byte) []byte
		cabeceras func(http.Header)
		max       int64
	}{
		{nombre: "ambito", mutar: func(r *ports.ResultadoConsultaOrganizacionHistorica) { r.Pagina.Selector.OrganismoRef = "org_ajena" }},
		{nombre: "corte", mutar: func(r *ports.ResultadoConsultaOrganizacionHistorica) {
			r.Pagina.Selector.ConocidoEn = r.Pagina.Selector.ConocidoEn.Add(time.Second)
		}},
		{nombre: "efecto", mutar: func(r *ports.ResultadoConsultaOrganizacionHistorica) { r.Evidencia.EfectoRef = "org_ajena" }},
		{nombre: "evidencia", mutar: func(r *ports.ResultadoConsultaOrganizacionHistorica) { r.Evidencia.DecisionRef = "" }},
		{nombre: "cursor", mutar: func(r *ports.ResultadoConsultaOrganizacionHistorica) { r.Pagina.CursorSiguiente = "repite" }},
		{nombre: "duplicadas", raw: func(b []byte) []byte { return []byte(`{"data":null,` + string(b[1:])) }},
		{nombre: "mayusculas", raw: func(b []byte) []byte { return []byte(strings.Replace(string(b), "decision_ref", "DECISION_REF", 1)) }},
		{nombre: "version_null", raw: func(b []byte) []byte {
			return []byte(strings.Replace(string(b), `"version_rpt_ref":""`, `"version_rpt_ref":null`, 1))
		}},
		{nombre: "desconocidas", raw: func(b []byte) []byte { return []byte(`{"privado":true,` + string(b[1:])) }},
		{nombre: "bytes", max: 100},
		{nombre: "cookie", cabeceras: func(h http.Header) { h.Set("Set-Cookie", "sesion=prohibida") }},
		{nombre: "compresion", cabeceras: func(h http.Header) { h.Set("Content-Encoding", "gzip") }},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			r := paginaPruebaOH(s)
			if tc.mutar != nil {
				tc.mutar(&r)
			}
			b := bodyPruebaOH(t, r)
			if tc.raw != nil {
				b = tc.raw(b)
			}
			cfg := fixtureTLSOH(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tc.cabeceras != nil {
					tc.cabeceras(w.Header())
				}
				if _, err := w.Write(b); err != nil {
					t.Error(err)
				}
			}))
			if tc.max != 0 {
				cfg.MaximoBytesPagina = tc.max
			}
			c, err := NuevoClienteOrganizacionHistorica(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Cerrar()
			out, err := c.Consultar(context.Background(), s)
			if err == nil || out.Evidencia.ReciboRef != "" {
				t.Fatal("respuesta manipulada publicada")
			}
		})
	}
}

func TestClienteOHNoSigueRedirectsNiCargaClaveConPermisoAmplio(t *testing.T) {
	s := selectorPruebaOH()
	var destino atomic.Int32
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destino.Add(1) }))
	defer h.Close()
	cfg := fixtureTLSOH(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, h.URL, 302) }))
	c, err := NuevoClienteOrganizacionHistorica(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Cerrar()
	if _, err := c.Consultar(context.Background(), s); err == nil || destino.Load() != 0 {
		t.Fatal("redirect seguido")
	}
	if err := os.Chmod(filepath.Join(cfg.Directorio, cfg.ClaveCliente), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoClienteOrganizacionHistorica(cfg); err == nil {
		t.Fatal("clave de lectura compartida")
	}
}
