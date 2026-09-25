package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorFormalizacionPrueba struct {
	recibida application.SolicitudDocumentacionFormalizacion
	llamadas int
	err      error
}

func (c *consultorFormalizacionPrueba) Consultar(_ context.Context, s application.SolicitudDocumentacionFormalizacion) (application.DocumentacionFormalizacion, error) {
	c.llamadas++
	c.recibida = s
	if c.err != nil {
		return application.DocumentacionFormalizacion{}, c.err
	}
	regla := ports.ReglaFormalizacion{Clave: "b21.plazo_documentacion", Unidad: "dias_habiles", Cantidad: 3, Computo: "administrativo",
		Origen: "ejemplo", Ejemplo: true, Norma: "Supuesto de trabajo.", Duda: "Duda 13.",
		Referencia: "vec.bolsa.reglas:1:b21.plazo_documentacion", HuellaCatalogo: strings.Repeat("a", 64)}
	return application.DocumentacionFormalizacion{
		ExpedienteDocumentalRef: "ref:" + strings.Repeat("9", 64), AceptadaEn: s.AceptadaEn, Documentos: []application.DocumentoExigidoFormalizacion{{Clave: "titulacion",
			TipoDocumental: "contratacion_temporal.formalizacion.titulacion.v1", Registrable: true}},
		ReglaDocumentos: regla,
		PlazoDocumentacion: application.PlazoFormalizacion{Regla: regla, Estado: application.PlazoFormalizacionUltimoDia,
			Vencimiento: ports.VencimientoFormalizacion{UltimoDia: "2026-09-30", VenceAntesDe: time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC)}},
		ConsultadaEn: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC),
	}, nil
}

const consultaFormalizacionPrueba = RutaDocumentacionFormalizacion + "?expediente_ref=expediente%3Act%3A001&aceptada_en=2026-09-25T09%3A05%3A00.12345Z"

func TestDocumentacionFormalizacionProyectaPlazosYProcedencia(t *testing.T) {
	consultor := &consultorFormalizacionPrueba{}
	h, err := NuevoManejadorDocumentacionFormalizacion(consultor)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba+"&modalidad=sustitucion", nil))
	cuerpo := w.Body.String()
	if w.Code != http.StatusOK || consultor.recibida.Modalidad != "sustitucion" || consultor.recibida.ExpedienteRef != "expediente:ct:001" ||
		!consultor.recibida.AceptadaEn.Equal(time.Date(2026, 9, 25, 9, 5, 0, 123450000, time.UTC)) {
		t.Fatalf("consulta=%d %s %+v", w.Code, cuerpo, consultor.recibida)
	}
	for _, esperado := range []string{`"esquema":"vec.ct.formalizacion.documentacion.v1"`, `"expediente_documental_ref":"ref:9999`, `"ultimo_dia":"2026-09-30"`,
		`"estado":"ultimo_dia"`, `"ejemplo":true`, `"referencia":"vec.bolsa.reglas:1:b21.plazo_documentacion"`,
		`"tipo_documental":"contratacion_temporal.formalizacion.titulacion.v1"`, `"registrable":true`} {
		if !strings.Contains(cuerpo, esperado) {
			t.Errorf("falta %s en %s", esperado, cuerpo)
		}
	}
	if strings.Contains(cuerpo, "plazo_incorporacion") || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("salida inesperada: %s", cuerpo)
	}
}

func TestDocumentacionFormalizacionRechazaPeticionesNoCanonicas(t *testing.T) {
	casos := map[string]*http.Request{
		"sin aceptacion":    httptest.NewRequest(http.MethodGet, RutaDocumentacionFormalizacion, nil),
		"parametro libre":   httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba+"&actor=x", nil),
		"repetido":          httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba+"&aceptada_en=2026-09-25T09:05:00Z", nil),
		"fecha no RFC 3339": httptest.NewRequest(http.MethodGet, RutaDocumentacionFormalizacion+"?aceptada_en=25/09/2026", nil),
		"modalidad vacia":   httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba+"&modalidad=", nil),
		"con cuerpo":        httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba, strings.NewReader("{}")),
	}
	for nombre, r := range casos {
		consultor := &consultorFormalizacionPrueba{}
		h, _ := NuevoManejadorDocumentacionFormalizacion(consultor)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || consultor.llamadas != 0 {
			t.Errorf("%s: %d llamadas=%d", nombre, w.Code, consultor.llamadas)
		}
	}
	h, _ := NuevoManejadorDocumentacionFormalizacion(&consultorFormalizacionPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, consultaFormalizacionPrueba, nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDocumentacionFormalizacion+"/otra?aceptada_en=2026-09-25T09:05:00Z", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("ruta ajena=%d", w.Code)
	}
}

func TestDocumentacionFormalizacionDistingueSinReglasDeNoDisponible(t *testing.T) {
	casos := map[error]struct {
		estado int
		codigo string
	}{
		ports.ErrReglasFormalizacionNoConfiguradas:   {http.StatusServiceUnavailable, "reglas_no_configuradas"},
		ports.ErrReglasFormalizacionNoDisponibles:    {http.StatusServiceUnavailable, "servicio_no_disponible"},
		ports.ErrSolicitudDocumentacionFormalizacion: {http.StatusUnprocessableEntity, "contenido_no_valido"},
		errors.New("interno con detalle privado"):    {http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for causa, esperado := range casos {
		h, _ := NuevoManejadorDocumentacionFormalizacion(&consultorFormalizacionPrueba{err: causa})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, consultaFormalizacionPrueba, nil))
		if w.Code != esperado.estado || !strings.Contains(w.Body.String(), `"codigo":"`+esperado.codigo+`"`) ||
			strings.Contains(w.Body.String(), "detalle privado") {
			t.Errorf("%v: %d %s", causa, w.Code, w.Body.String())
		}
	}
	if _, err := NuevoManejadorDocumentacionFormalizacion(nil); !errors.Is(err, ErrManejadorDocumentacionFormalizacionInvalido) {
		t.Fatal("consultor nulo aceptado")
	}
}
