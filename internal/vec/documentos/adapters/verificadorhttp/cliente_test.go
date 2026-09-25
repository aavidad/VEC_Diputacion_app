package verificadorhttp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func certificadoClientePrueba(t *testing.T) ([]byte, []byte) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "cliente-prueba"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "cliente-prueba"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	claveDER, err := x509.MarshalPKCS8PrivateKey(clave)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: claveDER})
}

func pruebaTLS(t *testing.T, handler http.HandlerFunc) (*httptest.Server, Configuracion) {
	t.Helper()
	servidor := httptest.NewUnstartedServer(handler)
	servidor.TLS = &tls.Config{MinVersion: tls.VersionTLS13, ClientAuth: tls.RequireAnyClientCert}
	servidor.StartTLS()
	t.Cleanup(servidor.Close)
	certificado, clave := certificadoClientePrueba(t)
	return servidor, Configuracion{
		URL:                   servidor.URL,
		CAPEM:                 pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servidor.Certificate().Raw}),
		CertificadoClientePEM: certificado, ClaveClientePEM: clave,
	}
}

func solicitudPrueba() ports.SolicitudVerificacionFirma {
	original := []byte("original conservado")
	huella := sha256.Sum256(original)
	return ports.SolicitudVerificacionFirma{
		DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 2,
		HuellaOriginalSHA256: hex.EncodeToString(huella[:]),
		ContenidoOriginal:    original,
		ContenidoFirmado:     []byte("artefacto firmado"),
	}
}

func respuestaPrueba(s ports.SolicitudVerificacionFirma, estado string) respuestaHTTP {
	huella := sha256.Sum256(s.ContenidoFirmado)
	return respuestaHTTP{
		VersionContrato: 1, Estado: estado,
		HuellaOriginalSHA256:    s.HuellaOriginalSHA256,
		HuellaFirmadoSHA256:     hex.EncodeToString(huella[:]),
		FirmanteRef:             "ref:" + strings.Repeat("2", 64),
		CertificadoHuellaSHA256: strings.Repeat("a", 64),
		SelloTiempoEstado:       "valido", RevocacionEstado: "vigente", VinculoOriginal: true,
	}
}

func TestVerificarTLSMutuoYVinculoExacto(t *testing.T) {
	s := solicitudPrueba()
	servidor, config := pruebaTLS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/verificaciones" || r.Method != http.MethodPost ||
			len(r.TLS.PeerCertificates) != 1 {
			t.Errorf("solicitud o autenticacion mutua inesperada")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var recibida solicitudHTTP
		if err := json.NewDecoder(r.Body).Decode(&recibida); err != nil {
			t.Error(err)
		}
		if recibida.HuellaOriginalSHA256 != s.HuellaOriginalSHA256 ||
			string(recibida.ContenidoOriginal) != string(s.ContenidoOriginal) ||
			string(recibida.ContenidoFirmado) != string(s.ContenidoFirmado) {
			t.Error("contenido o huella alterados")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respuestaPrueba(s, "valida"))
	})
	_ = servidor
	cliente, err := Nuevo(config)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := cliente.Verificar(context.Background(), s)
	if err != nil || resultado.Estado != ports.EstadoVerificacionValida || !resultado.VinculoOriginal {
		t.Fatalf("resultado = %+v, %v", resultado, err)
	}
}

func TestVerificarCierraAnteRespuestaNoVinculada(t *testing.T) {
	casos := []struct {
		name    string
		cambiar func(*respuestaHTTP)
	}{
		{"huella original", func(r *respuestaHTTP) { r.HuellaOriginalSHA256 = strings.Repeat("b", 64) }},
		{"huella firmado", func(r *respuestaHTTP) { r.HuellaFirmadoSHA256 = strings.Repeat("b", 64) }},
		{"sin vinculo", func(r *respuestaHTTP) { r.VinculoOriginal = false }},
		{"sin revocacion", func(r *respuestaHTTP) { r.RevocacionEstado = "indeterminada" }},
		{"sin sello", func(r *respuestaHTTP) { r.SelloTiempoEstado = "indeterminado" }},
		{"estado desconocido", func(r *respuestaHTTP) { r.Estado = "firmado" }},
	}
	for _, caso := range casos {
		t.Run(caso.name, func(t *testing.T) {
			s := solicitudPrueba()
			_, config := pruebaTLS(t, func(w http.ResponseWriter, _ *http.Request) {
				r := respuestaPrueba(s, "valida")
				caso.cambiar(&r)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(r)
			})
			cliente, err := Nuevo(config)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := cliente.Verificar(context.Background(), s)
			if !errors.Is(err, ErrRespuestaInvalida) || resultado.Estado == ports.EstadoVerificacionValida {
				t.Fatalf("respuesta insegura aceptada: %+v, %v", resultado, err)
			}
		})
	}
}

func TestVerificarNoValidaEIndeterminadaNoSeElevan(t *testing.T) {
	for _, estado := range []string{"no_valida", "indeterminada"} {
		t.Run(estado, func(t *testing.T) {
			s := solicitudPrueba()
			_, config := pruebaTLS(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				r := respuestaPrueba(s, estado)
				r.FirmanteRef = ""
				r.CertificadoHuellaSHA256 = ""
				r.VinculoOriginal = false
				_ = json.NewEncoder(w).Encode(r)
			})
			cliente, err := Nuevo(config)
			if err != nil {
				t.Fatal(err)
			}
			r, err := cliente.Verificar(context.Background(), s)
			if err != nil || string(r.Estado) != estado {
				t.Fatalf("resultado: %+v, %v", r, err)
			}
		})
	}
}

func TestVerificarFallaCerradoEnEntradaYRedireccion(t *testing.T) {
	s := solicitudPrueba()
	_, config := pruebaTLS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://otro.example/v1/verificaciones", http.StatusTemporaryRedirect)
	})
	cliente, err := Nuevo(config)
	if err != nil {
		t.Fatal(err)
	}
	s.ContenidoOriginal[0] ^= 1
	if _, err = cliente.Verificar(context.Background(), s); !errors.Is(err, ports.ErrVerificacionFirmaInvalida) {
		t.Fatal(err)
	}
	s = solicitudPrueba()
	if _, err = cliente.Verificar(context.Background(), s); !errors.Is(err, ErrVerificacionNoDisponible) {
		t.Fatal(err)
	}
	config.URL = "http://localhost"
	if _, err = Nuevo(config); !errors.Is(err, ErrConfiguracion) {
		t.Fatal(err)
	}
}

func TestVerificarLimitaRespuestaYTiempo(t *testing.T) {
	s := solicitudPrueba()
	t.Run("respuesta excesiva", func(t *testing.T) {
		_, config := pruebaTLS(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(strings.Repeat("x", maximaRespuesta+1)))
		})
		cliente, err := Nuevo(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cliente.Verificar(context.Background(), s); !errors.Is(err, ErrRespuestaInvalida) {
			t.Fatal(err)
		}
	})
	t.Run("tiempo agotado", func(t *testing.T) {
		_, config := pruebaTLS(t, func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{}"))
		})
		config.Timeout = 30 * time.Millisecond
		cliente, err := Nuevo(config)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cliente.Verificar(context.Background(), s); !errors.Is(err, ErrVerificacionNoDisponible) {
			t.Fatal(err)
		}
	})
}
