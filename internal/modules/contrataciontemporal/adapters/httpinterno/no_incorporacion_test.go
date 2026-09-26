package httpinterno

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorNoIncorporacionPrueba struct {
	ejecutorGINPIXPrueba
	no application.SolicitudRegistrarNoIncorporacion
}

func (e *ejecutorNoIncorporacionPrueba) RegistrarNoIncorporacion(_ context.Context, s application.SolicitudRegistrarNoIncorporacion) (ports.ReciboOperacionSeguimiento, error) {
	e.no = s
	r, err := e.recibo(ports.OperacionRegistrarNoIncorporacion)
	r.FaseResultante, r.CausaClave, r.FechaEfecto = domain.FaseFiscalizacion, domain.ClaveCatalogo(s.MotivoClave), ""
	return r, err
}
func (e *ejecutorNoIncorporacionPrueba) Opciones(ctx context.Context) (ports.OpcionesSeguimiento, error) {
	o, err := e.ejecutorGINPIXPrueba.Opciones(ctx)
	o.NoIncorporacion = &ports.ReglaNoIncorporacion{SegundaPersona: true, Motivos: []ports.MotivoNoIncorporacion{
		{Clave: "no_presentado", Etiqueta: "No se presenta", ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo"}}}
	return o, err
}

func TestNoIncorporacionHTTPConSusConflictos(t *testing.T) {
	sin, _ := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, &ejecutorSeguimientoPrueba{})
	if _, ok := sin[RutaNoIncorporaciones]; ok {
		t.Fatal("sin la no incorporación compuesta no hay ruta")
	}
	e := &ejecutorNoIncorporacionPrueba{ejecutorGINPIXPrueba: ejecutorGINPIXPrueba{conGIN: true, estado: ports.EstadoSeguimientoExpediente{
		ExpedienteRef: "expediente:prueba", Acreditada: &ports.EstadoIncorporacionAcreditada{NoIncorporacion: &ports.EstadoNoIncorporacion{
			MotivoClave: "no_presentado", ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo", ResolucionRef: "resolucion:rrhh:1",
			ResolucionSHA256: strings.Repeat("b", 64), ResueltaPor: "per_segunda", FechaNotificacion: "2026-09-20", ReciboRef: "recibo:ni:1",
			RegistradaEn: time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)},
			PropuestaNoIncorporacion: &ports.EstadoPropuestaNoIncorporacion{PropuestaRef: "recibo:ni:propuesta", MotivoClave: "no_presentado",
				ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo", ResolucionRef: "resolucion:rrhh:2", ResolucionSHA256: strings.Repeat("c", 64),
				FechaNotificacion: "2026-09-22", RegistradaEn: time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)}}}}}
	m, err := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, e)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":7,"clave_idempotencia":"44444444-4444-4444-8444-444444444444","paso":"confirmar","propuesta_ref":"recibo:ni:propuesta","motivo_clave":"no_presentado","resolucion_ref":"resolucion:rrhh:1","resolucion_sha256":"` +
		strings.Repeat("b", 64) + `","resuelta_por":"","fecha_notificacion":"2026-09-20","observaciones":""}`
	w := peticionSeguimiento(t, m, RutaNoIncorporaciones, cuerpo)
	if w.Code != http.StatusCreated || e.no.MotivoClave != "no_presentado" || e.no.Paso != "confirmar" || e.no.PropuestaRef != "recibo:ni:propuesta" ||
		!e.no.FechaNotificacion.Equal(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)) ||
		!strings.Contains(w.Body.String(), `"operacion":"registrar_no_incorporacion"`) || !strings.Contains(w.Body.String(), `"fase_resultante":"fiscalizacion"`) {
		t.Fatalf("no incorporación: %d %s", w.Code, w.Body.String())
	}
	if w := peticionSeguimiento(t, m, RutaNoIncorporaciones, strings.Replace(cuerpo, `"motivo_clave"`, `"motivo"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("campo desconocido: %d", w.Code)
	}
	for err, codigo := range map[error]string{ports.ErrSinAceptacion: "sin_aceptacion", ports.ErrIncorporacionExistente: "incorporacion_existente",
		ports.ErrNoIncorporacionExistente: "no_incorporacion_existente", ports.ErrFechaNoIncorporacionNoAdmitida: "fecha_no_admitida",
		ports.ErrPropuestaNoIncorporacionPendiente: "propuesta_pendiente", ports.ErrPropuestaNoIncorporacionNoValida: "propuesta_no_valida",
		ports.ErrMismaPersonaNoIncorporacion: "misma_persona"} {
		e.error = err
		w := peticionSeguimiento(t, m, RutaNoIncorporaciones, cuerpo)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"codigo":"`+codigo+`"`) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	e.error = nil
	w = peticionSeguimiento(t, m, RutaSeguimientoCese, `{"expediente_ref":"expediente:prueba"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"segunda_persona":true`) || !strings.Contains(w.Body.String(), `"clave":"no_presentado"`) ||
		!strings.Contains(w.Body.String(), `"resuelta_por":"per_segunda"`) || strings.Contains(w.Body.String(), "b24.sancion") ||
		!strings.Contains(w.Body.String(), `"no_incorporacion_propuesta":{`) || !strings.Contains(w.Body.String(), `"propuesta_ref":"recibo:ni:propuesta"`) {
		t.Fatalf("consulta con la no incorporación: %d %s", w.Code, w.Body.String())
	}
}
