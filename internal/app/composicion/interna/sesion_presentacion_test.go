package interna

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadcertificado"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registradorFronteraSesionPrueba struct {
	ordenes []ports.OrdenFronteraIdentidadTecnica
	falla   bool
}

func (r *registradorFronteraSesionPrueba) RegistrarRechazoInicioSesion(
	_ context.Context, orden ports.OrdenFronteraIdentidadTecnica,
) (ports.AcuseFronteraIdentidadTecnica, error) {
	r.ordenes = append(r.ordenes, orden)
	if r.falla {
		return ports.AcuseFronteraIdentidadTecnica{}, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	return ports.AcuseFronteraIdentidadTecnica{
		AuditoriaRef: "aud_v3_pit_0123456789abcdef0123456789abcdef",
		Secuencia:    1, MaterialSHA256: strings.Repeat("a", 64),
		CorrelacionRef: orden.CorrelacionRef(), RegistradaEn: time.Now().UTC(),
	}, nil
}

func TestSesionC4RechazoEsperaAcuseTecnico(t *testing.T) {
	registrador := &registradorFronteraSesionPrueba{}
	p := &puenteConsultaSeguimiento{
		extractor:               &extractorCertificadoPersonalDirecto{},
		api:                     http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("API alcanzada") }),
		presentacionCertificado: &httpseguridad.ServicioPresentacionCertificado{},
		auditorSesion:           registrador,
	}
	p.fachada.Store(&FachadaIdentidadOffline{propietario: &tokenServidorInterno{marca: 1}})
	responder := func(r *http.Request) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		p.ServeHTTP(w, r)
		return w
	}
	if w := responder(httptest.NewRequest(http.MethodGet, domain.RutaInicioSesion, nil)); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("método sin rechazo confirmado: %d", w.Code)
	}
	if len(registrador.ordenes) != 1 {
		t.Fatal("rechazo de método sin orden técnica")
	}
	datos, err := registrador.ordenes[0].Datos()
	if err != nil || datos.MetodoEsperado != http.MethodPost || datos.Ruta != domain.RutaInicioSesion ||
		datos.Motivo != ports.MotivoFronteraIdentidadMetodoNoPermitido {
		t.Fatalf("orden técnica incorrecta: %v", err)
	}
	registrador.falla = true
	if w := responder(httptest.NewRequest(http.MethodPost, domain.RutaSesionActual, nil)); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("auditoría caída permitió denegar como éxito: %d", w.Code)
	}
	if err := p.AuditarRechazoSesionNoCanonica(context.Background(), domain.RutaInicioSesion); err == nil {
		t.Fatal("alias sin auditoría confirmado")
	}
	registrador.falla = false
	for _, cabecera := range []string{"X-Proxy-User", "X-VEC-Perfil"} {
		peticion := httptest.NewRequest(http.MethodGet, domain.RutaSesionActual, nil)
		peticion.Header.Set(cabecera, "ajeno")
		if w := responder(peticion); w.Code != http.StatusBadRequest {
			t.Fatalf("cabecera %s admitida: %d", cabecera, w.Code)
		}
		datos, err := registrador.ordenes[len(registrador.ordenes)-1].Datos()
		if err != nil || datos.Motivo != ports.MotivoFronteraIdentidadSolicitudInvalida {
			t.Fatalf("cabecera %s sin auditoría de solicitud inválida: %v", cabecera, err)
		}
	}
	p.presentacionCertificado = nil
	p.auditorSesion = nil
	if w := responder(httptest.NewRequest(http.MethodGet, domain.RutaSesionActual, nil)); w.Code != http.StatusNotFound {
		t.Fatalf("material C4 ausente cambió la ruta heredada: %d", w.Code)
	}
}

func TestAsercionC4InicioPOSTExacto(t *testing.T) {
	intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
	certificado := intercambio.estadoServidor.VerifiedChains[0][0]
	huella := sha256.Sum256(certificado.Raw)
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	escribirCRLPrueba(t, directorio, intercambio, nil)
	ruta := filepath.Join(directorio, "certificados.json")
	escribirRegistroCertificadoPrueba(t, ruta, certificadoPersonalRegistrado{
		HuellaSHA256:       "sha256:" + hex.EncodeToString(huella[:]),
		SujetoID:           "per_aaaaaaaaaaaaaaaaaaaaaa",
		CuentaID:           "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11,
		Activo:             true,
	})
	registro, err := identidadcertificado.NuevoRegistro(ruta)
	if err != nil {
		t.Fatal(err)
	}
	cfg := configuracionInternaValidaPrueba()
	cfg.RetiradaPoliticaInternaEn = time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	_, firmante, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	extractor, verificador, _, err := nuevaAsercionCertificadoPersonal(cfg,
		"clave_desarrollo_01", firmante, registro,
		"pga_certificado_desarrollo_protegido_01", "sha256:"+strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	preparar := func(metodo, ruta string) *http.Request {
		t.Helper()
		r := httptest.NewRequest(metodo, ruta, nil)
		estado := intercambio.estadoServidor
		r.TLS = &estado
		r = r.WithContext(context.WithValue(r.Context(), claveContextoCanalTLSInterno{},
			nuevaCapacidadCanalTLSInterno(&tokenServidorInterno{marca: 1}, estado)))
		preparada, err := httpseguridad.PrepararPeticionAsercionPasarela(r, 1024)
		if err != nil {
			t.Fatal(err)
		}
		return preparada
	}
	inicio := preparar(http.MethodPost, "/api/vec/session/start")
	asercion, err := extractor.ExtraerAsercionProtegida(inicio)
	if err != nil || len(asercion) == 0 {
		t.Fatalf("POST explícito no emitió aserción: %v", err)
	}
	defer clear(asercion)
	verificada, err := verificador.Verificar(inicio.Context(), asercion)
	if err != nil || verificada.SujetoID != "per_aaaaaaaaaaaaaaaaaaaaaa" ||
		verificada.Cuenta.ID != "cta_bbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("aserción POST inválida: %v", err)
	}
	if _, err := verificador.Verificar(preparar(http.MethodGet, "/api/vec/session/start").Context(), asercion); err == nil {
		t.Fatal("aserción POST reutilizada como GET")
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodPost, "/api/vec/session"},
		{http.MethodPost, "/api/vec/session/start?actor=ajeno"},
		{http.MethodPost, "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento"},
	} {
		if _, err := extractor.ExtraerAsercionProtegida(preparar(caso.metodo, caso.ruta)); err == nil {
			t.Fatalf("POST ajeno admitido: %s", caso.ruta)
		}
	}
	cuerpo := inicio.Clone(inicio.Context())
	const datosCliente = `{"actor":"ajeno","perfil":"ajeno","nonce":"cliente"}`
	cuerpo.Body = io.NopCloser(strings.NewReader(datosCliente))
	cuerpo.ContentLength = int64(len(datosCliente))
	if _, err := extractor.ExtraerAsercionProtegida(cuerpo); err == nil {
		t.Fatal("cuerpo JSON aportado por cliente admitido")
	}
	inicio.Header.Set("Authorization", "Bearer ajeno")
	if _, err := extractor.ExtraerAsercionProtegida(inicio); err == nil {
		t.Fatal("Bearer ajeno admitido")
	}
	inicio.Header.Del("Authorization")
	inicio.Header.Set("Cookie", "sesion=ajena")
	if _, err := extractor.ExtraerAsercionProtegida(inicio); err == nil {
		t.Fatal("cookie admitida")
	}
	inicio.Header.Del("Cookie")
	escribirCRLPrueba(t, directorio, intercambio, certificado.SerialNumber)
	if _, err := extractor.ExtraerAsercionProtegida(inicio); err == nil {
		t.Fatal("certificado revocado admitido en POST")
	}
}
