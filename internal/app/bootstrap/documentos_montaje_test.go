package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type registradorDocumentosPrueba struct {
	ordenes []docpg.OrdenDenegacionFrontera
	err     error
}

func (r *registradorDocumentosPrueba) RegistrarDenegacion(_ context.Context, o docpg.OrdenDenegacionFrontera) error {
	if o.Validar() != nil {
		return docpg.ErrOrdenFronteraInvalida
	}
	r.ordenes = append(r.ordenes, o)
	return r.err
}

type incidenciasDocumentosPrueba struct {
	emitidas []core.SolicitudIncidenciaTecnica
}

func (i *incidenciasDocumentosPrueba) Emitir(s core.SolicitudIncidenciaTecnica) {
	i.emitidas = append(i.emitidas, s)
}

func manifiestoIncluido(manifiestos []core.ModuleManifest, id string) bool {
	for _, m := range manifiestos {
		if m.ID == id {
			return true
		}
	}
	return false
}

func TestDocumentosApagadoPorDefectoNoCambiaNada(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	a, err := nuevosDocumentosDesarrollo(cfg, nil, nil, nil, io.Discard)
	if err != nil || a != nil {
		t.Fatal("con el selector apagado no debe componerse nada", err)
	}
	if manifiestoIncluido(manifiestosShellVEC(cfg), "vec.module.documentos") {
		t.Fatal("el catálogo ofrece Documentos con el selector apagado")
	}
	cfg.DocumentosEnabled = "true"
	if !manifiestoIncluido(manifiestosShellVEC(cfg), "vec.module.documentos") {
		t.Fatal("el catálogo no ofrece Documentos con el montaje activado")
	}
	cfg.DevelopmentGuard = ""
	if manifiestoIncluido(manifiestosShellVEC(cfg), "vec.module.documentos") {
		t.Fatal("el catálogo ofrece Documentos fuera de la doble llave")
	}
}

func TestDocumentosActivadoFallaCerradoSinMaterial(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.DocumentosEnabled = "si"
	if _, err := nuevosDocumentosDesarrollo(cfg, nil, nil, nil, io.Discard); !errors.Is(err, config.ErrConfiguracionDocumentosSelector) {
		t.Fatal("selector no canónico aceptado", err)
	}
	cfg.DocumentosEnabled = "true"
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevosDocumentosDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, nil, io.Discard); !errors.Is(err, ErrComposicionDocumentosNoDisponible) {
		t.Fatal("arranca sin material V3 de Documentos", err)
	}
	proveedor := &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	if _, err := nuevosDocumentosDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, proveedor, io.Discard); !errors.Is(err, ErrComposicionDocumentosNoDisponible) {
		t.Fatal("arranca sin configuración privada de Documentos", err)
	}
	// Material presente pero con DSN inservibles: tampoco arranca y el error
	// no revela DSN ni rutas.
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", ficheroMaterialDocumentos)
	material := `{"version":1,"autoridad":"no_autoritativo","cuentas":[],"motivos":{"listar":{}},"almacen":{"tipo":"ficheros"}}`
	if err := os.WriteFile(ruta, []byte(material), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = nuevosDocumentosDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, proveedor, io.Discard)
	if !errors.Is(err, ErrComposicionDocumentosNoDisponible) || strings.Contains(err.Error(), cfg.DevelopmentMaterialDir) {
		t.Fatal("material incompleto aceptado o error con ruta privada", err)
	}
	cfg.DevelopmentGuard = ""
	if _, err := nuevosDocumentosDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, proveedor, io.Discard); !errors.Is(err, config.ErrConfiguracionDocumentosActivacion) {
		t.Fatal("arranca fuera de la doble llave", err)
	}
}

// Arranque completo sobre PostgreSQL real de Contratación (se omite sin él):
// apagado, el servidor arranca y /api/vec/documentos/ no existe; activado sin
// material privado de Documentos, falla cerrado con ErrComposicionDocumentos.
func TestServidorDesarrolloConDocumentosApagadoYActivadoSinMaterial(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, _, err := NewHTTPServerDesarrolloWithConfig(cfg, io.Discard)
	if err != nil || servidor == nil {
		t.Fatalf("el arranque apagado cambió: %v", err)
	}
	w := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(w, peticionDocumentos(docpg.RutaFronteraConsulta, false))
	if w.Code == http.StatusOK {
		t.Fatal("ruta documental servida con el selector apagado")
	}
	cfg.DocumentosEnabled = "true"
	servidor, _, err = NewHTTPServerDesarrolloWithConfig(cfg, io.Discard)
	if servidor != nil || !errors.Is(err, ErrComposicionDocumentosNoDisponible) {
		t.Fatalf("activado sin material: servidor=%v err=%v", servidor != nil, err)
	}
}

// Sin PostgreSQL de Contratación tampoco arranca con el selector activado.
func TestServidorDesarrolloConDocumentosActivadoSinGobiernoNoArranca(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.DocumentosEnabled = "true"
	if servidor, _, err := NewHTTPServerDesarrolloWithConfig(cfg, io.Discard); servidor != nil || err == nil {
		t.Fatal("el servidor arrancó con Documentos activado y sin gobierno V3")
	}
}

func peticionDocumentos(ruta string, conTLS bool) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	if conTLS {
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}}}}
	}
	return r
}

func TestFronteraDocumentosAuditaDenegacionesYFallaCerrado(t *testing.T) {
	registrador := &registradorDocumentosPrueba{}
	incidencias := &incidenciasDocumentosPrueba{}
	a := &autoridadDocumentosDesarrollo{base: &autoridadRutasDietasDesarrollo{}, reloj: relojRutasDietas{}, registrador: registrador,
		incidencias: incidencias, publicadas: map[string]bool{docpg.RutaFronteraConsulta: true}}
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := a.proteger(siguiente)

	// Fuera del prefijo documental, la frontera no interviene.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/vec/session", nil))
	if w.Code != http.StatusTeapot {
		t.Fatalf("ruta ajena alterada: %d", w.Code)
	}
	// Sin mTLS: 401 sin escritura durable; ruta no publicada: 404.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionDocumentos(docpg.RutaFronteraConsulta, false))
	if w.Code != http.StatusUnauthorized || len(registrador.ordenes) != 0 || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("sin mTLS: %d %d", w.Code, len(registrador.ordenes))
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionDocumentos("/api/vec/documentos/originales/descargas", false))
	if w.Code != http.StatusNotFound || len(registrador.ordenes) != 0 {
		t.Fatalf("descarga no publicada: %d", w.Code)
	}
	// Con identidad TLS: cookie o cabecera libre se deniegan y se auditan.
	r := peticionDocumentos(docpg.RutaFronteraConsulta, true)
	r.Header.Set("Cookie", "sesion=robada")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || len(registrador.ordenes) != 1 || registrador.ordenes[0].Motivo != docpg.MotivoFronteraAutenticacion ||
		registrador.ordenes[0].Ruta != docpg.RutaFronteraConsulta || strings.Contains(w.Body.String(), "robada") {
		t.Fatalf("cookie: %d %+v", w.Code, registrador.ordenes)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionDocumentos("/api/vec/documentos/originales/descargas", true))
	if w.Code != http.StatusNotFound || registrador.ordenes[1].Ruta != docpg.RutaFronteraOtra {
		t.Fatalf("ruta no publicada con TLS: %d %+v", w.Code, registrador.ordenes)
	}
	// Si la auditoría no se confirma: 503 e incidencia técnica.
	registrador.err = errors.New("auditor caído")
	r = peticionDocumentos(docpg.RutaFronteraConsulta, true)
	r.Header.Set("Authorization", "Bearer x")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || len(incidencias.emitidas) != 1 ||
		incidencias.emitidas[0].Codigo != core.IncidenciaAuditoriaNoRegistrada {
		t.Fatalf("auditoría caída: %d %+v", w.Code, incidencias.emitidas)
	}
}

func TestAutoridadExactasConDocumentosExigeContextoDeLaMismaPeticion(t *testing.T) {
	a := &autoridadDocumentosDesarrollo{reloj: relojRutasDietas{}}
	cadena := autoridadExactasConDocumentos{documentos: a}
	if err := cadena.AutorizarRutaExacta(context.Background(), docpg.RutaFronteraConsulta); !errors.Is(err, vechttp.ErrAccesoRutaExactaDenegado) {
		t.Fatalf("ruta documental sin contexto: %v", err)
	}
	ctx := context.WithValue(context.Background(), claveContextoDocumentos{}, contextoDocumentos{autoridad: &autoridadDocumentosDesarrollo{}, ruta: docpg.RutaFronteraConsulta})
	if err := cadena.AutorizarRutaExacta(ctx, docpg.RutaFronteraConsulta); !errors.Is(err, vechttp.ErrAccesoRutaExactaDenegado) {
		t.Fatalf("contexto de otra autoridad aceptado: %v", err)
	}
	if err := cadena.AutorizarRutaExacta(context.Background(), "/api/vec/contratacion-temporal/x"); err == nil {
		t.Fatal("ruta ajena sin autoridad delegada aceptada")
	}
	consulta := autoridadConsultaDocumentos{autoridad: a, emisor: emisorCronosEmpleadoDesarrollo{}}
	if _, err := consulta.ResolverConsultaExpediente(context.Background(), docports.ConsultaExpediente{ExpedienteRef: "ref:" + strings.Repeat("2", 64), Limite: 5}); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("consulta sin identidad de la frontera: %v", err)
	}
}

func TestDescriptorMaterialDocumentosEsUnicoYNominal(t *testing.T) {
	d := descriptoresMaterialDocumentosDesarrollo()
	if len(d) != 1 || d[0].Audiencia != docports.AudienciaV3 {
		t.Fatalf("descriptor documental: %+v", d)
	}
	todos := append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), descriptoresMaterialCronosDesarrollo()...)
	todos = append(todos, descriptoresMaterialDietasDesarrollo()...)
	todos = append(todos, descriptoresMaterialPersonalB2Desarrollo()...)
	todos = append(todos, d...)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("el descriptor documental colisiona con otro consumidor", err)
	}
}
