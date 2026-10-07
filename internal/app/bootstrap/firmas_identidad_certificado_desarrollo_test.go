package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type relojCertificadoFirmaVecPOSTPrueba struct{ ahora time.Time }

func (r *relojCertificadoFirmaVecPOSTPrueba) Ahora() time.Time { return r.ahora }

type acreditadorCertificadoFirmaVecPOSTPrueba struct {
	respuesta AcreditacionCertificadoFirmaVecV2
	llamadas  int
	err       error
}

func (a *acreditadorCertificadoFirmaVecPOSTPrueba) AcreditarCertificadoFirmaVecV2(
	_ context.Context, hoja, ca *x509.Certificate, _ time.Time,
) (AcreditacionCertificadoFirmaVecV2, error) {
	a.llamadas++
	if hoja == nil || ca == nil {
		return AcreditacionCertificadoFirmaVecV2{}, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return a.respuesta, a.err
}

type emisorCertificadoFirmaVecPOSTPrueba struct {
	identidad httpseguridad.AsercionProxyIdentidad
	vinculo   httpseguridad.VinculoPeticionPasarela
	llamadas  int
}

func (e *emisorCertificadoFirmaVecPOSTPrueba) Emitir(_ context.Context,
	a httpseguridad.AsercionProxyIdentidad, v httpseguridad.VinculoPeticionPasarela,
) ([]byte, error) {
	e.llamadas++
	e.identidad, e.vinculo = a, v
	return []byte("asercion-protegida-de-prueba"), nil
}

func TestExtractorCertificadoFirmaVecPOSTUsaCanalYCuerpoPreparado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	reloj := &relojCertificadoFirmaVecPOSTPrueba{ahora: ahora}
	acreditador := &acreditadorCertificadoFirmaVecPOSTPrueba{respuesta: AcreditacionCertificadoFirmaVecV2{
		PersonaRef: "per_0123456789abcdefghijkl", CuentaRef: "cta_0123456789abcdefghijkl"}}
	emisor := &emisorCertificadoFirmaVecPOSTPrueba{}
	e := &extractorCertificadoFirmaVecV2{emisor: emisor, acreditador: acreditador,
		emisorID: "https://idp.example.invalid", audiencia: "vec-interna",
		retirada: ahora.Add(time.Hour), reloj: reloj}
	cuerpo := []byte(`{"firmado_base64":"JVBERi0xLjcKJSVFT0Y="}`)
	r := httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, strings.NewReader(string(cuerpo)))
	estado := estadoTLSRealFirmanteV2Prueba(t)
	r.TLS = &estado
	vinculo, err := httpseguridad.NuevoVinculoPeticionPasarela(r.Method, r.URL.RequestURI(), cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := e.extraer(r, vinculo)
	if err != nil || string(resultado) != "asercion-protegida-de-prueba" ||
		acreditador.llamadas != 1 || emisor.llamadas != 1 || emisor.vinculo != vinculo {
		t.Fatalf("extracción POST: %v", err)
	}
	if emisor.identidad.SujetoID != acreditador.respuesta.PersonaRef ||
		emisor.identidad.Cuenta.ID != acreditador.respuesta.CuentaRef ||
		emisor.identidad.MetodoPrimario != httpseguridad.MetodoCertificado ||
		emisor.identidad.ACRVerificado != httpseguridad.ACRCertificadoPersonalDesarrolloProtegido ||
		len(emisor.identidad.Factores) != 1 || emisor.identidad.CanalVinculadoRef == "" {
		t.Fatal("identidad de certificado incompleta")
	}
	contenido, err := io.ReadAll(r.Body)
	if err != nil || string(contenido) != string(cuerpo) {
		t.Fatal("el extractor modificó el cuerpo firmado", err)
	}
	r.Header.Set("Authorization", "Bearer inyectado")
	if _, err := e.extraer(r, vinculo); !errors.Is(err, errIdentidadCertificadoFirmaVecNoDisponible) ||
		acreditador.llamadas != 1 || emisor.llamadas != 1 {
		t.Fatal("cabecera ambiental llegó a autoridades", err)
	}
	r.Header.Del("Authorization")
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: estado.PeerCertificates, VerifiedChains: estado.VerifiedChains}
	if _, err := e.extraer(r, vinculo); !errors.Is(err, errIdentidadCertificadoFirmaVecNoDisponible) ||
		acreditador.llamadas != 2 || emisor.llamadas != 1 {
		t.Fatal("certificado sin canal exportable produjo aserción", err)
	}
}

type auditoriaCertificadoFirmaVecPOSTPrueba struct{ llamadas int }

func (a *auditoriaCertificadoFirmaVecPOSTPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context,
	orden vp.OrdenAuditoriaFronteraRutaExacta,
) error {
	a.llamadas++
	if orden.Validar() != nil || orden.Ruta != httpinterno.RutaRegistroFirmaVec {
		return errors.New("orden inesperada")
	}
	return nil
}

func TestPuenteCertificadoFirmaVecPOSTCierraRutaMetodoYOrigen(t *testing.T) {
	auditoria := &auditoriaCertificadoFirmaVecPOSTPrueba{}
	llamadas := 0
	p := &puenteCertificadoFirmaVecV2{entorno: &EntornoIdentidadCertificadoFirmaVecV2{
		origen: "https://vec.example.invalid", host: "vec.example.invalid", auditoria: auditoria},
		siguiente: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })}
	casos := []struct {
		nombre, metodo, ruta, origen string
		estado                       int
	}{
		{"ruta_ajena", http.MethodPost, httpinterno.RutaRegistroFirmaExterna, "https://vec.example.invalid", 404},
		{"metodo_ajeno", http.MethodGet, httpinterno.RutaRegistroFirmaVec, "https://vec.example.invalid", 405},
		{"origen_ajeno", http.MethodPost, httpinterno.RutaRegistroFirmaVec, "https://otro.example.invalid", 403},
		{"sin_origen", http.MethodPost, httpinterno.RutaRegistroFirmaVec, "", 403},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := httptest.NewRequest(c.metodo, c.ruta, nil)
			r.Host = "vec.example.invalid"
			if c.origen != "" {
				r.Header.Set("Origin", c.origen)
			}
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)
			if w.Code != c.estado || llamadas != 0 || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("frontera %d, llamadas %d", w.Code, llamadas)
			}
		})
	}
	if auditoria.llamadas != 2 {
		t.Fatalf("denegaciones de origen sin auditoría: %d", auditoria.llamadas)
	}
}
