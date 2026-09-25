package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func consultaFormalizacionDesarrollo(t *testing.T, ruta http.Handler, aceptada time.Time) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	ruta.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		httpinterno.RutaDocumentacionFormalizacion+"?expediente_ref=expediente%3Act%3A001&aceptada_en="+url.QueryEscape(aceptada.Format(time.RFC3339Nano)), nil))
	return w
}

func TestDocumentacionFormalizacionUsaElCatalogoDeBolsaYLaConservacion(t *testing.T) {
	aceptada := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	madrid, _ := time.LoadLocation("Europe/Madrid")
	ultimo, err := calendariosdomain.FechaCivilDe(aceptada.In(madrid).AddDate(0, 0, 5))
	if err != nil {
		t.Fatal(err)
	}
	finDia, err := ultimo.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	calendarios := &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo: calendariosdomain.ResultadoPlazo{Vencimiento: ultimo, VenceAntesDe: finDia},
	}}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), calendarios, relojCalendariosDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	ruta, err := nuevaRutaDocumentacionFormalizacionDesarrollo(compuestas.bolsa)
	if err != nil || !rutaDocumentacionFormalizacionDesarrollo(ruta.Ruta) {
		t.Fatalf("ruta: %v", err)
	}
	w := consultaFormalizacionDesarrollo(t, ruta.Manejador, aceptada)
	cuerpo := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("consulta=%d %s", w.Code, cuerpo)
	}
	for _, clave := range []string{"documento_identidad", "titulacion", "declaracion_no_separacion", "declaracion_compatibilidad", "certificado_delitos_sexuales"} {
		if !strings.Contains(cuerpo, `{"clave":"`+clave+`","tipo_documental":"contratacion_temporal.formalizacion.`+clave+`.v1","registrable":true}`) {
			t.Errorf("documento %s sin tipo catalogado: %s", clave, cuerpo)
		}
	}
	for _, esperado := range []string{`"clave":"b21.plazo_documentacion"`, `"clave":"b23.plazo_incorporacion"`,
		`"ultimo_dia":"` + ultimo.String() + `"`, `"articulo":"art. 11.1"`, `"estado":"en_curso"`} {
		if !strings.Contains(cuerpo, esperado) {
			t.Errorf("falta %s: %s", esperado, cuerpo)
		}
	}
	if !calendarios.recibida.NotificadoEn.Equal(aceptada) || calendarios.recibida.Cantidad != 1 {
		t.Fatalf("el cómputo no parte de la aceptación: %+v", calendarios.recibida)
	}
}

func TestDocumentacionFormalizacionSinCatalogoNoSuponePlazos(t *testing.T) {
	ruta, err := nuevaRutaDocumentacionFormalizacionDesarrollo(nil)
	if err != nil {
		t.Fatal(err)
	}
	w := consultaFormalizacionDesarrollo(t, ruta.Manejador, time.Now().UTC().Add(-time.Hour))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"codigo":"reglas_no_configuradas"`) {
		t.Fatalf("sin catálogo=%d %s", w.Code, w.Body.String())
	}
	// Con catálogo pero sin Calendarios, el plazo hábil no se inventa.
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojCalendariosDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	ruta, _ = nuevaRutaDocumentacionFormalizacionDesarrollo(compuestas.bolsa)
	w = consultaFormalizacionDesarrollo(t, ruta.Manejador, time.Now().UTC().Add(-time.Hour))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"codigo":"servicio_no_disponible"`) {
		t.Fatalf("sin Calendarios=%d %s", w.Code, w.Body.String())
	}
}
