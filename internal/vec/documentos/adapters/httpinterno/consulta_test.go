package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type autoridadPrueba struct {
	denegar    bool
	errorFijo  error
	recurso    string
	expediente string
	limite     uint32
	llamadas   int
}

func (a *autoridadPrueba) resultado() (ports.AutorizacionV3, error) {
	if a.errorFijo != nil {
		return ports.AutorizacionV3{}, a.errorFijo
	}
	if a.denegar {
		return ports.AutorizacionV3{}, errors.New("denegado")
	}
	return ports.AutorizacionV3{}, nil
}
func (a *autoridadPrueba) ResolverConsultaExpediente(_ context.Context, c ports.ConsultaExpediente) (ports.AutorizacionV3, error) {
	a.recurso, a.limite = c.ExpedienteRef, c.Limite
	a.llamadas++
	return a.resultado()
}
func (a *autoridadPrueba) ResolverDescargaOriginal(_ context.Context, c ports.ConsultaDocumento, expediente string) (ports.AutorizacionV3, error) {
	a.recurso, a.expediente = c.DocumentoID, expediente
	a.llamadas++
	return a.resultado()
}

type servicioPrueba struct {
	documento    domain.Documento
	original     ports.Original
	cursor       string
	ultimoCursor string
	llamadas     int
	err          error
}

func (s *servicioPrueba) ListarExpediente(_ context.Context, consulta ports.ConsultaExpediente) (ports.PaginaDocumentos, error) {
	s.llamadas++
	s.ultimoCursor = consulta.Cursor
	if s.err != nil {
		return ports.PaginaDocumentos{}, s.err
	}
	return ports.PaginaDocumentos{Items: []domain.Documento{s.documento}, SiguienteCursor: s.cursor}, nil
}
func (s *servicioPrueba) DescargarOriginal(_ context.Context, _ ports.ConsultaDocumento) (ports.Original, error) {
	s.llamadas++
	if s.err != nil {
		return ports.Original{}, s.err
	}
	return s.original, nil
}

type incidenciasPrueba struct {
	emitidas []vecdomain.SolicitudIncidenciaTecnica
}

func (i *incidenciasPrueba) Emitir(s vecdomain.SolicitudIncidenciaTecnica) {
	i.emitidas = append(i.emitidas, s)
}
func documentoPrueba() domain.Documento {
	return domain.Documento{
		ID: "ref:" + strings.Repeat("1", 64), NumeroVEC: "VEC-2026-1", ModuloID: "dietas", ExpedienteRef: "ref:" + strings.Repeat("2", 64),
		TipoRef: "ref:" + strings.Repeat("3", 64), Version: 1, MIME: "application/pdf", HuellaSHA256: strings.Repeat("a", 64), Tamano: 6,
		ObjetoRef: "obj:123", ObjetoVersion: "version:1", PoliticaRef: "ref:" + strings.Repeat("4", 64), VersionPolitica: 1,
		HuellaPoliticaSHA256: strings.Repeat("b", 64), ConservacionHasta: time.Now().Add(time.Hour),
		Proteccion: "conservacion", EstadoPolitica: domain.EstadoPoliticaAprobada, EstadoFirma: domain.EstadoFirmaPendienteProveedor, CreadoEn: time.Now(), Custodia: domain.CustodiaVEC,
	}
}
func solicitud(ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func TestRutasListaExigenContextoYNoAceptanCabecerasLibres(t *testing.T) {
	s := &servicioPrueba{documento: documentoPrueba()}
	a := &autoridadPrueba{denegar: true}
	rutas, err := NuevasRutasExactas(s, a)
	if err != nil || len(rutas) != 2 {
		t.Fatal(err)
	}
	r := solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","limite":50}`)
	r.Header.Set("X-VEC-Subject", "actor_inventado")
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || s.llamadas != 0 || a.recurso != "ref:2222222222222222222222222222222222222222222222222222222222222222" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("denegacion=%d servicio=%d", w.Code, s.llamadas)
	}
	a.denegar = false
	w = httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","limite":50,"actor":"inventado"}`))
	if w.Code != http.StatusUnprocessableEntity || s.llamadas != 0 {
		t.Fatalf("cuerpo libre=%d servicio=%d", w.Code, s.llamadas)
	}
	w = httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente+"?actor=inventado", `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","limite":50}`))
	if w.Code != http.StatusNotFound || s.llamadas != 0 {
		t.Fatalf("ruta no canonica=%d", w.Code)
	}
	w = httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","limite":50}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"numero_vec":"VEC-2026-1"`) || strings.Contains(w.Body.String(), "obj:123") {
		t.Fatalf("lista=%d %s", w.Code, w.Body.String())
	}
}
func TestDescargaConservaBytesYVerificaHuella(t *testing.T) {
	pdf := []byte("%PDF-1.7\noriginal")
	suma := sha256.Sum256(pdf)
	huella := hex.EncodeToString(suma[:])
	s := &servicioPrueba{original: ports.Original{Contenido: pdf, MIME: "application/pdf", HuellaSHA256: huella}}
	a := &autoridadPrueba{}
	rutas, _ := NuevasRutasExactas(s, a)
	w := httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, solicitud(RutaDescargaOriginal, `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","documento_ref":"ref:1111111111111111111111111111111111111111111111111111111111111111","version":1}`))
	if w.Code != http.StatusOK || w.Body.String() != string(pdf) || w.Header().Get("X-Content-SHA256") != huella ||
		w.Header().Get("Content-Disposition") != `attachment; filename="documento-ref_1111111111111111111111111111111111111111111111111111111111111111.pdf"` || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("descarga=%d cabeceras=%v", w.Code, w.Header())
	}
	s.original.HuellaSHA256 = strings.Repeat("a", 64)
	w = httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, solicitud(RutaDescargaOriginal, `{"expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","documento_ref":"ref:1111111111111111111111111111111111111111111111111111111111111111","version":1}`))
	if w.Code != http.StatusBadGateway || w.Header().Get("Content-Disposition") != "" {
		t.Fatalf("huella falsa=%d", w.Code)
	}
}
func TestConstructorCierraDependenciasNulas(t *testing.T) {
	if _, err := NuevasRutasExactas(nil, &autoridadPrueba{}); !errors.Is(err, ErrManejadorInvalido) {
		t.Fatal(err)
	}
}

func TestListaMarcaCustodiaExternaSinDescargaNiReferenciaDelCustodio(t *testing.T) {
	d := documentoPrueba()
	d.ObjetoRef, d.ObjetoVersion, d.MIME, d.Tamano = "", "", "application/pdf", 0
	d.Custodia = domain.CustodiaExterna
	d.CustodiaExternaRef = domain.ReferenciaCustodiaExterna{CustodioID: "dietas.justificantes", Referencia: "justificante:interno:0001", HuellaSHA256: d.HuellaSHA256}
	s := &servicioPrueba{documento: d}
	rutas, _ := NuevasRutasExactas(s, &autoridadPrueba{})
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:`+strings.Repeat("2", 64)+`","limite":50}`))
	cuerpo := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(cuerpo, `"custodia":"externa"`) || !strings.Contains(cuerpo, `"descargable":false`) ||
		strings.Contains(cuerpo, "justificante:interno") || strings.Contains(cuerpo, "dietas.justificantes") {
		t.Fatalf("lista externa=%d %s", w.Code, cuerpo)
	}
}

func TestListaTransportaCursorSoloEnCuerpoYRechazaBucle(t *testing.T) {
	s := &servicioPrueba{documento: documentoPrueba(), cursor: "cursor:primera"}
	rutas, _ := NuevasRutasExactas(s, &autoridadPrueba{})
	ref := "ref:" + strings.Repeat("2", 64)
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"`+ref+`","limite":50}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"siguiente_cursor":"cursor:primera"`) {
		t.Fatalf("pagina inicial=%d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"`+ref+`","cursor":"cursor:primera","limite":50}`))
	if w.Code != http.StatusBadGateway || s.ultimoCursor != "cursor:primera" {
		t.Fatalf("cursor repetido=%d", w.Code)
	}
}

func TestErroresDelServicioSeTraducenSinDetalleInterno(t *testing.T) {
	ref := "ref:" + strings.Repeat("2", 64)
	casos := []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrConflicto, http.StatusConflict, "conflicto"},
		{ports.ErrValidacion, http.StatusUnprocessableEntity, "contenido_no_valido"},
		{ports.ErrNoEncontrado, http.StatusNotFound, "recurso_no_encontrado"},
		{ports.ErrAccesoDenegado, http.StatusForbidden, "acceso_denegado"},
		{errors.Join(ports.ErrCapacidadNoDisponible, errors.New("dial tcp 10.0.0.1:5432 secreto")), http.StatusServiceUnavailable, "servicio_no_disponible"},
		{errors.New("pq: texto interno"), http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, c := range casos {
		inc := &incidenciasPrueba{}
		s := &servicioPrueba{documento: documentoPrueba(), err: c.err}
		rutas, err := NuevasRutasExactasConIncidencias(s, &autoridadPrueba{}, inc)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"`+ref+`","limite":10}`))
		cuerpo := w.Body.String()
		if w.Code != c.estado || !strings.Contains(cuerpo, `"codigo":"`+c.codigo+`"`) || strings.Contains(cuerpo, "secreto") || strings.Contains(cuerpo, "interno") {
			t.Fatalf("%v: %d %s", c.err, w.Code, cuerpo)
		}
		if (c.estado >= 500) != (len(inc.emitidas) == 1) {
			t.Fatalf("%v: incidencias %+v", c.err, inc.emitidas)
		}
		if c.estado >= 500 && inc.emitidas[0].Codigo != vecdomain.IncidenciaHTTPInternoFallido {
			t.Fatalf("incidencia inesperada %+v", inc.emitidas[0])
		}
	}
}

func TestAutoridadCaidaEsIndisponibilidadYDenegacionEs403(t *testing.T) {
	ref := "ref:" + strings.Repeat("2", 64)
	inc := &incidenciasPrueba{}
	a := &autoridadPrueba{errorFijo: errors.Join(ports.ErrCapacidadNoDisponible, errors.New("gobierno caído"))}
	s := &servicioPrueba{documento: documentoPrueba()}
	rutas, _ := NuevasRutasExactasConIncidencias(s, a, inc)
	w := httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, solicitud(RutaDescargaOriginal, `{"expediente_ref":"`+ref+`","documento_ref":"ref:`+strings.Repeat("1", 64)+`","version":2}`))
	if w.Code != http.StatusServiceUnavailable || s.llamadas != 0 || a.expediente != ref || len(inc.emitidas) != 1 {
		t.Fatalf("autoridad caída=%d servicio=%d expediente=%q", w.Code, s.llamadas, a.expediente)
	}
	a.errorFijo = errors.New("sin concesión")
	w = httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, solicitud(RutaDescargaOriginal, `{"expediente_ref":"`+ref+`","documento_ref":"ref:`+strings.Repeat("1", 64)+`","version":2}`))
	if w.Code != http.StatusForbidden || len(inc.emitidas) != 1 {
		t.Fatalf("denegación=%d", w.Code)
	}
	w = httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, solicitud(RutaDescargaOriginal, `{"documento_ref":"ref:`+strings.Repeat("1", 64)+`","version":2}`))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("descarga sin expediente=%d", w.Code)
	}
}

func TestSinDescargaNoSePublicaRutaNiSeOfrecenBytes(t *testing.T) {
	s := &servicioPrueba{documento: documentoPrueba()}
	rutas, err := NuevasRutas(Configuracion{Servicio: s, Autoridad: &autoridadPrueba{}})
	if err != nil || len(rutas) != 1 || rutas[0].Ruta != RutaConsultaExpediente {
		t.Fatalf("rutas sin descarga: %v %d", err, len(rutas))
	}
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:`+strings.Repeat("2", 64)+`","limite":5}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"descargable":false`) {
		t.Fatalf("lista sin descarga=%d %s", w.Code, w.Body.String())
	}
}

type tiposPrueba map[string]string

func (t tiposPrueba) ClaveTipo(ref string) (string, bool) { c, ok := t[ref]; return c, ok }

func TestListaDeclaraClaveDeTipoYNuncaLaReferenciaOpaca(t *testing.T) {
	d := documentoPrueba()
	s := &servicioPrueba{documento: d}
	rutas, _ := NuevasRutas(Configuracion{Servicio: s, Autoridad: &autoridadPrueba{}, Tipos: tiposPrueba{d.TipoRef: "dietas.comision.borrador.v1"}})
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:`+strings.Repeat("2", 64)+`","limite":5}`))
	if !strings.Contains(w.Body.String(), `"tipo":"dietas.comision.borrador.v1"`) || strings.Contains(w.Body.String(), d.TipoRef) {
		t.Fatalf("tipo catalogado: %s", w.Body.String())
	}
	rutas, _ = NuevasRutas(Configuracion{Servicio: s, Autoridad: &autoridadPrueba{}, Tipos: tiposPrueba{}})
	w = httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:`+strings.Repeat("2", 64)+`","limite":5}`))
	if !strings.Contains(w.Body.String(), `"tipo":"documento"`) || strings.Contains(w.Body.String(), d.TipoRef) {
		t.Fatalf("tipo desconocido: %s", w.Body.String())
	}
}

func TestListaDeclaraConservacionProvisional(t *testing.T) {
	d := documentoPrueba()
	d.EstadoPolitica = domain.EstadoPoliticaProvisional
	rutas, _ := NuevasRutas(Configuracion{Servicio: &servicioPrueba{documento: d}, Autoridad: &autoridadPrueba{}})
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, solicitud(RutaConsultaExpediente, `{"expediente_ref":"ref:`+strings.Repeat("2", 64)+`","limite":5}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"conservacion":"provisional"`) {
		t.Fatalf("conservación provisional: %d %s", w.Code, w.Body.String())
	}
}
