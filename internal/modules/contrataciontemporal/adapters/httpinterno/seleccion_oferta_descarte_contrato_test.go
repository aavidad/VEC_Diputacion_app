package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func reciboDescarteOfertaHTTPPrueba() application.DatosReciboSeleccionLlamamientoParaAdaptador {
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{Descarte: &application.DatosReciboDescarteOfertaParaAdaptador{
		Esquema: "vec.contratacion-temporal.recibo-descarte-oferta.v1", Estado: "descartada",
		OrganizacionRef: "organizacion:http:001", ExpedienteRef: "expediente:http:001",
		ReciboCierreBolsaRef: "recibo:cierre:001", ReciboDescarteRef: "recibo:descarte:001",
		DescartadaEn: time.Date(2026, 10, 15, 12, 0, 0, 123456000, time.UTC),
	}}
}

func peticionRecuperarOfertaHTTPPrueba(accion string) *http.Request {
	cuerpo := `{"expediente_ref":"expediente:http:001","version_esperada":6,"clave_idempotencia":"` + claveSeleccionLlamamientoHTTPPrueba + `","accion":"` + accion + `"}`
	r := httptest.NewRequest(http.MethodPost, RutaSeleccionLlamamiento, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json")
	return r
}

func TestSeleccionOfertaHTTPRecuperacionExplicitaMinimizaDescarte(t *testing.T) {
	e := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: reciboDescarteOfertaHTTPPrueba()}
	w := httptest.NewRecorder()
	nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, e).ServeHTTP(w, peticionRecuperarOfertaHTTPPrueba("recuperar_oferta"))
	if w.Code != http.StatusOK {
		t.Fatalf("recuperación: %d %s", w.Code, w.Body)
	}
	q, ok := e.ultima()
	if !ok || !q.RecuperarOferta || q.ClaveIdempotencia != claveSeleccionLlamamientoHTTPPrueba || q.VersionEsperada != 6 {
		t.Fatalf("no conserva la intención: %+v", q)
	}
	var cuerpo struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if len(cuerpo.Data) != 7 || string(cuerpo.Data["estado"]) != `"descartada"` || cuerpo.Data["oferta_ref"] != nil || cuerpo.Data["bolsa_ref"] != nil {
		t.Fatalf("proyección no minimizada: %s", w.Body)
	}
}

func TestSeleccionOfertaHTTPDescarteNoEsRespuestaSingularNiOferta(t *testing.T) {
	for _, nombre := range []string{"peticion ordinaria", "resultado mezclado", "expediente ajeno", "fecha sin microsegundos"} {
		t.Run(nombre, func(t *testing.T) {
			r := reciboDescarteOfertaHTTPPrueba()
			peticion := peticionRecuperarOfertaHTTPPrueba("recuperar_oferta")
			switch nombre {
			case "peticion ordinaria":
				peticion = nuevaPeticionSeleccionLlamamientoHTTPPrueba()
			case "resultado mezclado":
				r.ReciboRef = "recibo:llamamiento:001"
			case "expediente ajeno":
				r.Descarte.ExpedienteRef = "expediente:ajeno:001"
			case "fecha sin microsegundos":
				r.Descarte.DescartadaEn = r.Descarte.DescartadaEn.Add(time.Nanosecond)
			}
			w := httptest.NewRecorder()
			nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: r}).ServeHTTP(w, peticion)
			if w.Code != http.StatusBadGateway {
				t.Fatalf("resultado inválido aceptado: %d %s", w.Code, w.Body)
			}
		})
	}
	for _, accion := range []string{"", "publicar", "cerrar_sin_comprobar"} {
		e := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: reciboDescarteOfertaHTTPPrueba()}
		w := httptest.NewRecorder()
		nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, e).ServeHTTP(w, peticionRecuperarOfertaHTTPPrueba(accion))
		if w.Code < 400 || e.total() != 0 {
			t.Fatalf("acción no canónica llegó al ejecutor: %q %d", accion, w.Code)
		}
	}
}

func TestSeleccionOfertaHTTPPriorizaResultadoIncierto(t *testing.T) {
	err := errors.Join(application.ErrSeleccionOfertaIndeterminada, application.ErrResultadoSeleccionOfertaNoConfiable)
	p, ok := clasificarErrorSeleccionOferta(err)
	if !ok || p.estado != http.StatusServiceUnavailable {
		t.Fatalf("efecto ambiguo no conserva recuperación: %+v", p)
	}
	for _, err := range []error{application.ErrSeleccionOfertaDescartada, application.ErrSeleccionOfertaSinReserva} {
		p, ok := clasificarErrorSeleccionOferta(err)
		if !ok || p.estado != http.StatusConflict {
			t.Fatalf("estado terminal sin conflicto: %+v", p)
		}
	}
}
