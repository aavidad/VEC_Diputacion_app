package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorCancelacionPrueba struct {
	solicitud application.SolicitudCancelarExpediente
	error     error
	estado    ports.EstadoCancelacionExpediente
}

func (e *ejecutorCancelacionPrueba) CancelarExpediente(_ context.Context, s application.SolicitudCancelarExpediente) (ports.ReciboOperacionSeguimiento, error) {
	e.solicitud = s
	return ports.ReciboOperacionSeguimiento{Operacion: ports.OperacionCancelarExpediente, OrganizacionRef: "organizacion:prueba", ExpedienteRef: s.ExpedienteRef,
		VersionAnterior: 3, VersionResultante: 4, FaseResultante: "asignacion_unidad", EstadoResultante: domain.EstadoCancelado,
		ReciboRef: "recibo:prueba", AuditoriaRef: "aud_v3_" + strings.Repeat("0", 32), EventoRef: "evento:prueba", ActorRef: "per_prueba",
		RegistradaEn: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC), MotivoClave: s.MotivoClave}, e.error
}
func (e *ejecutorCancelacionPrueba) Opciones(context.Context) (application.OpcionesCancelacion, error) {
	return application.OpcionesCancelacion{Fases: []domain.ClaveFase{"solicitud", "asignacion_unidad"},
		Motivos: []ports.MotivoCancelacion{{Clave: "necesidad_desaparecida", Etiqueta: "Ha desaparecido la necesidad", Canales: []domain.CanalCancelacion{"rrhh"}}},
		Todos: []ports.MotivoCancelacion{{Clave: "necesidad_desaparecida", Etiqueta: "Ha desaparecido la necesidad", Canales: []domain.CanalCancelacion{"rrhh"}},
			{Clave: "desistimiento_centro", Etiqueta: "El centro retira su petición", Canales: []domain.CanalCancelacion{"centro"}}}}, nil
}
func (e *ejecutorCancelacionPrueba) Estado(_ context.Context, _, exp string) (ports.EstadoCancelacionExpediente, error) {
	e.estado.ExpedienteRef = exp
	return e.estado, nil
}

func TestCancelacionHTTPRegistraYTraduceRechazos(t *testing.T) {
	ejecutor := &ejecutorCancelacionPrueba{}
	m, err := NuevosManejadoresCancelacion(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":3,"clave_idempotencia":"44444444-4444-4444-8444-444444444444","motivo_clave":"necesidad_desaparecida","observaciones":"Ya no hace falta"}`
	w := peticionSeguimiento(t, m, RutaCancelacionesExpediente, cuerpo)
	if w.Code != http.StatusCreated || ejecutor.solicitud.Canal.OrganizacionRef != "organizacion:prueba" || ejecutor.solicitud.Observaciones != "Ya no hace falta" {
		t.Fatalf("cancelación: %d %s", w.Code, w.Body.String())
	}
	var salida struct {
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Data["estado_resultante"] != "cancelado" || salida.Data["motivo_clave"] != "necesidad_desaparecida" {
		t.Fatalf("recibo: %s", w.Body.String())
	}
	if w := peticionSeguimiento(t, m, RutaCancelacionesExpediente, strings.Replace(cuerpo, `"observaciones"`, `"actor_ref":"per_x","observaciones"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("campo ajeno: %d", w.Code)
	}
	for err, codigo := range map[error]string{ports.ErrCancelacionTrasFiscalizacion: "tras_fiscalizacion", ports.ErrCancelacionNoAdmitida: "fase_no_admitida",
		ports.ErrCancelacionYaRegistrada: "cancelacion_existente", domain.ErrVersionEnConflicto: "version_en_conflicto"} {
		ejecutor.error = err
		if w := peticionSeguimiento(t, m, RutaCancelacionesExpediente, cuerpo); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"`+codigo+`"`) ||
			!strings.Contains(w.Body.String(), "api.contratacion_temporal.cancelacion.error."+codigo) {
			t.Fatalf("%s: %d %s", codigo, w.Code, w.Body.String())
		}
	}
	ejecutor.error = ports.ErrAutorizacionDenegada
	if w := peticionSeguimiento(t, m, RutaCancelacionesExpediente, cuerpo); w.Code != http.StatusForbidden {
		t.Fatalf("denegada: %d", w.Code)
	}
}

func TestCancelacionHTTPConsultaOpcionesYEstado(t *testing.T) {
	ejecutor := &ejecutorCancelacionPrueba{}
	m, _ := NuevosManejadoresCancelacion(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, ejecutor)
	w := peticionSeguimiento(t, m, RutaCancelacionExpediente, `{"expediente_ref":"expediente:prueba"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"fases_admitidas":["solicitud","asignacion_unidad"]`) ||
		!strings.Contains(w.Body.String(), `"cancelacion":null`) || strings.Contains(w.Body.String(), "canales") {
		t.Fatalf("consulta: %d %s", w.Code, w.Body.String())
	}
	ejecutor.estado.Cancelacion = &ports.CancelacionRegistrada{Canal: "centro", MotivoClave: "desistimiento_centro", FasePrevia: "solicitud",
		ReciboRef: "recibo:prueba", RegistradaEn: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)}
	if w := peticionSeguimiento(t, m, RutaCancelacionExpediente, `{"expediente_ref":"expediente:prueba"}`); !strings.Contains(w.Body.String(), `"motivo_clave":"desistimiento_centro"`) ||
		!strings.Contains(w.Body.String(), `"motivo_etiqueta":"El centro retira su petición"`) {
		t.Fatalf("consulta con cancelación: %s", w.Body.String())
	}
	denegada, _ := NuevosManejadoresCancelacion(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{lecturaDenegada: true}, ejecutor)
	if w := peticionSeguimiento(t, denegada, RutaCancelacionExpediente, `{"expediente_ref":"expediente:prueba"}`); w.Code != http.StatusForbidden {
		t.Fatalf("lectura denegada: %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodGet, RutaCancelacionExpediente, nil)
	w = httptest.NewRecorder()
	m[RutaCancelacionExpediente].ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", w.Code)
	}
}
