package administracion

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
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

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestServidorAdminSoloCertificadoDeCAPropiaNoRevocado(t *testing.T) {
	ahora := time.Now()
	caClave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caModelo := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA ADMIN prueba"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, SubjectKeyId: []byte{1, 2, 3},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caModelo, caModelo, &caClave.PublicKey, caClave)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	crearHoja := func(serie int64, uso x509.ExtKeyUsage) (tls.Certificate, []byte, []byte) {
		clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		modelo := &x509.Certificate{
			SerialNumber: big.NewInt(serie), Subject: pkix.Name{CommonName: "identidad sintetica"},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso},
			DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		}
		der, err := x509.CreateCertificate(rand.Reader, modelo, ca, &clave.PublicKey, caClave)
		if err != nil {
			t.Fatal(err)
		}
		claveDER, err := x509.MarshalECPrivateKey(clave)
		if err != nil {
			t.Fatal(err)
		}
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		clavePEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: claveDER})
		par, err := tls.X509KeyPair(certPEM, clavePEM)
		if err != nil {
			t.Fatal(err)
		}
		return par, certPEM, clavePEM
	}
	_, servidorPEM, clavePEM := crearHoja(2, x509.ExtKeyUsageServerAuth)
	cliente, _, _ := crearHoja(3, x509.ExtKeyUsageClientAuth)
	hoja, err := x509.ParseCertificate(cliente.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if !cadenaDirectaVigente([]*x509.Certificate{hoja, ca}, ca, ahora) {
		t.Fatal("cadena directa valida rechazada")
	}
	if cadenaDirectaVigente([]*x509.Certificate{hoja, ca, ca}, ca, ahora) {
		t.Fatal("CA intermedia aceptada")
	}
	if cadenaDirectaVigente([]*x509.Certificate{hoja, ca}, ca, ahora.Add(2*time.Hour)) {
		t.Fatal("conexion antigua conserva hoja caducada")
	}
	dir := t.TempDir()
	escribir := func(nombre string, datos []byte) string {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, datos, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	rutaCRL := filepath.Join(dir, "admin.crl")
	actualizarCRL := func(revocar bool) {
		lista := &x509.RevocationList{Number: big.NewInt(1), ThisUpdate: ahora.Add(-time.Minute), NextUpdate: ahora.Add(time.Hour)}
		if revocar {
			lista.RevokedCertificateEntries = []x509.RevocationListEntry{{SerialNumber: big.NewInt(3), RevocationTime: ahora}}
		}
		der, err := x509.CreateRevocationList(rand.Reader, lista, ca, caClave)
		if err != nil {
			t.Fatal(err)
		}
		escribir("admin.crl", pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}))
	}
	actualizarCRL(false)
	cfg := Configuracion{
		Entorno: "cidonia", Escucha: "127.0.0.1:19443", Host: "admin.example.test",
		Audiencia: "vec-admin-prueba", EmisorIdentidad: "https://identidad.example.test",
		CertificadoServidor: escribir("servidor.crt", servidorPEM),
		ClaveServidor:       escribir("servidor.key", clavePEM),
		CAAdministracion:    escribir("ca.crt", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		CRLAdministracion:   rutaCRL, RedesPermitidas: []string{"0.0.0.0/0", "::/0"},
		RetiradaEn: ahora.Add(time.Hour).UTC().Truncate(time.Second),
	}
	servidor, err := NuevoServidor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	defer prueba.Close()
	peticion := func(path, host string, certificado bool) (int, error) {
		configTLS := &tls.Config{RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}
		if certificado {
			configTLS.Certificates = []tls.Certificate{cliente}
		}
		transporte := &http.Transport{TLSClientConfig: configTLS}
		defer transporte.CloseIdleConnections()
		req, _ := http.NewRequest(http.MethodGet, prueba.URL+path, nil)
		req.Host = host
		resp, err := (&http.Client{Transport: transporte}).Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		return resp.StatusCode, nil
	}
	if got, err := peticion("/livez", cfg.Host, true); err != nil || got != http.StatusNoContent {
		t.Fatalf("mTLS valido: %d %v", got, err)
	}
	if got, _ := peticion("/api/admin/estado", cfg.Host, true); got != http.StatusNotFound {
		t.Fatalf("ruta de negocio publicada: %d", got)
	}
	if got, _ := peticion("/livez", "otro.example.test", true); got != http.StatusForbidden {
		t.Fatalf("host ajeno: %d", got)
	}
	// VEC_ADMIN_HOST con puerto público: la cabecera Host debe llevar
	// exactamente ese puerto; «:443» equivale al nombre solo.
	for _, caso := range []struct {
		configurado string
		aceptados   []string
		rechazados  []string
	}{
		{"admin.example.test:8444", []string{"admin.example.test:8444"}, []string{"admin.example.test", "admin.example.test:443", "admin.example.test:8443", "admin.example.test:08444", "otro.example.test:8444"}},
		{"admin.example.test:443", []string{"admin.example.test"}, []string{"admin.example.test:443", "admin.example.test:8444"}},
		{"admin.example.test", []string{"admin.example.test"}, []string{"admin.example.test:443", "admin.example.test:8444"}},
	} {
		cfgPuerto := cfg
		cfgPuerto.Host = caso.configurado
		servidorPuerto, err := NuevoServidor(cfgPuerto)
		if err != nil {
			t.Fatalf("%s: %v", caso.configurado, err)
		}
		pruebaPuerto := httptest.NewUnstartedServer(servidorPuerto.Handler)
		pruebaPuerto.TLS = servidorPuerto.TLSConfig.Clone()
		pruebaPuerto.StartTLS()
		original := prueba.URL
		prueba.URL = pruebaPuerto.URL
		for _, host := range caso.aceptados {
			if got, err := peticion("/livez", host, true); err != nil || got != http.StatusNoContent {
				t.Fatalf("%s: host %s rechazado: %d %v", caso.configurado, host, got, err)
			}
		}
		for _, host := range caso.rechazados {
			if got, _ := peticion("/livez", host, true); got != http.StatusForbidden {
				t.Fatalf("%s: host %s admitido: %d", caso.configurado, host, got)
			}
		}
		prueba.URL = original
		pruebaPuerto.Close()
	}
	// ObservarADMIN con puerto público: exige la autoridad exacta en Host y
	// entrega el nombre sin puerto para host_admin y la autoridad aparte.
	cfgObservada := cfg
	cfgObservada.Host = "admin.example.test:8444"
	servidorObservado, err := NuevoServidor(cfgObservada)
	if err != nil {
		t.Fatal(err)
	}
	redObservada, err := httpseguridad.NuevaPoliticaRed(superficieSesionPerfiles(cfgObservada))
	if err != nil {
		t.Fatal(err)
	}
	hostObservado, _ := analizarHostAdmin(cfgObservada.Host)
	relojObservado := relojActivosPrueba{ahora: time.Now().UTC()}
	resolvedor := &resolvedorSesionPerfiles{cfg: cfgObservada, host: hostObservado, ca: ca.Raw, red: redObservada, reloj: relojObservado}
	contextoObservado, err := NuevoContextoConexionPerfiles(relojObservado)
	if err != nil {
		t.Fatal(err)
	}
	pruebaObservada := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o, err := resolvedor.ObservarADMIN(r.Context(), r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Prueba-Host", o.Host)
		w.Header().Set("X-Prueba-Autoridad", o.Autoridad)
		w.WriteHeader(http.StatusNoContent)
	}))
	pruebaObservada.Config.ConnContext = contextoObservado
	pruebaObservada.TLS = servidorObservado.TLSConfig.Clone()
	pruebaObservada.StartTLS()
	observar := func(configurado string, casos map[string]int) {
		t.Helper()
		cfgCaso := cfgObservada
		cfgCaso.Host = configurado
		hostCaso, _ := analizarHostAdmin(configurado)
		resolvedor = &resolvedorSesionPerfiles{cfg: cfgCaso, host: hostCaso, ca: ca.Raw, red: redObservada, reloj: relojObservado}
		for host, esperado := range casos {
			transporte := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cliente}}}
			req, _ := http.NewRequest(http.MethodGet, pruebaObservada.URL+"/", nil)
			req.Host = host
			resp, err := (&http.Client{Transport: transporte}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			transporte.CloseIdleConnections()
			if resp.StatusCode != esperado {
				t.Fatalf("ObservarADMIN %s con Host %s: %d, esperado %d", configurado, host, resp.StatusCode, esperado)
			}
			if esperado == http.StatusNoContent && (resp.Header.Get("X-Prueba-Host") != "admin.example.test" || resp.Header.Get("X-Prueba-Autoridad") != hostCaso.autoridad) {
				t.Fatalf("observación con host %q y autoridad %q", resp.Header.Get("X-Prueba-Host"), resp.Header.Get("X-Prueba-Autoridad"))
			}
		}
	}
	observar("admin.example.test:8444", map[string]int{"admin.example.test:8444": http.StatusNoContent, "admin.example.test": http.StatusUnauthorized, "admin.example.test:443": http.StatusUnauthorized, "admin.example.test:8443": http.StatusUnauthorized})
	observar("admin.example.test:443", map[string]int{"admin.example.test": http.StatusNoContent, "admin.example.test:443": http.StatusUnauthorized, "admin.example.test:8444": http.StatusUnauthorized})
	pruebaObservada.Close()
	for _, invalido := range []string{"Admin.Example.Test", "admin.example.test:0", "admin.example.test:65536", "admin.example.test:08444", "admin.example.test:", ":8444", "admin.example.test:8444:1", "[::1]:8444", "admin.example.test.", " admin.example.test"} {
		cfgInvalida := cfg
		cfgInvalida.Host = invalido
		if _, err := NuevoServidor(cfgInvalida); !errors.Is(err, ErrConfiguracion) {
			t.Fatalf("host %q admitido: %v", invalido, err)
		}
	}
	if _, err := peticion("/livez", cfg.Host, false); err == nil {
		t.Fatal("TLS acepto cliente sin certificado")
	}
	var llamadasPerfiles atomic.Int64
	correlaciones := make(chan string, 2)
	perfilHandler := &handlerPerfilesADMIN{auditor: &auditorActivosPrueba{}, api: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		canon, err := ref.ValorCanonico()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		correlaciones <- canon
		llamadasPerfiles.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}), rutas: map[string]string{}}
	servidorPerfiles, err := nuevoServidor(cfg, perfilHandler)
	if err != nil {
		t.Fatal(err)
	}
	pruebaPerfiles := httptest.NewUnstartedServer(servidorPerfiles.Handler)
	pruebaPerfiles.TLS = servidorPerfiles.TLSConfig.Clone()
	pruebaPerfiles.StartTLS()
	defer pruebaPerfiles.Close()
	peticionPerfil := func() int {
		transporte := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cliente}}}
		defer transporte.CloseIdleConnections()
		req, err := http.NewRequest(http.MethodGet, pruebaPerfiles.URL+"/api/admin/perfiles/v1/personas", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = cfg.Host
		req.Header.Set("X-Correlation-ID", strings.Repeat("a", 32))
		respuesta, err := (&http.Client{Transport: transporte}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer respuesta.Body.Close()
		return respuesta.StatusCode
	}
	if got := peticionPerfil(); got != http.StatusAccepted || llamadasPerfiles.Load() != 1 {
		t.Fatalf("perfil tras frontera=%d llamadas=%d", got, llamadasPerfiles.Load())
	}
	primera := <-correlaciones
	if got := peticionPerfil(); got != http.StatusAccepted || llamadasPerfiles.Load() != 2 {
		t.Fatalf("segunda petición=%d llamadas=%d", got, llamadasPerfiles.Load())
	}
	segunda := <-correlaciones
	if primera == segunda || primera == "correlacion_"+strings.Repeat("a", 32) || segunda == "correlacion_"+strings.Repeat("a", 32) {
		t.Fatal("correlacion_de_cliente_o_reutilizada")
	}
	actualizarCRL(true)
	if got := peticionPerfil(); got != http.StatusForbidden || llamadasPerfiles.Load() != 2 {
		t.Fatalf("revocado llegó al handler=%d llamadas=%d", got, llamadasPerfiles.Load())
	}
	if got, _ := peticion("/livez", cfg.Host, true); got != http.StatusForbidden {
		t.Fatalf("revocacion: %d", got)
	}
	if err := comprobarCertificadoVigente(hoja, ca, rutaCRL, ahora); !errors.Is(err, errCertificadoRevocado) {
		t.Fatalf("revocacion sin causa nominal: %v", err)
	}
	if err := os.Remove(rutaCRL); err != nil {
		t.Fatal(err)
	}
	if err := comprobarCertificadoVigente(hoja, ca, rutaCRL, ahora); !errors.Is(err, errCRLNoDisponible) {
		t.Fatalf("CRL ausente sin causa nominal: %v", err)
	}
	if got, _ := peticion("/livez", cfg.Host, true); got != http.StatusForbidden {
		t.Fatalf("CRL ausente: %d", got)
	}
	escribir("admin.crl", []byte("invalida"))
	if err := comprobarCertificadoVigente(hoja, ca, rutaCRL, ahora); !errors.Is(err, errCRLInvalida) {
		t.Fatalf("CRL invalida sin causa nominal: %v", err)
	}
	cfg.Entorno = "produccion"
	if _, err := NuevoServidor(cfg); !errors.Is(err, ErrConfiguracion) {
		t.Fatal("produccion sin Kerberos arranco")
	}
	cfg.Entorno = "cidonia"
	cfg.CertificadoServidor = filepath.Join(dir, "certificado-privado-inexistente.crt")
	_, err = NuevoServidor(cfg)
	var rutaErr *os.PathError
	if !errors.Is(err, ErrConfiguracion) || !errors.As(err, &rutaErr) {
		t.Fatalf("causa TLS no conservada: %v", err)
	}
	if strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), "certificado-privado") {
		t.Fatal("ruta privada filtrada en diagnostico")
	}
}
