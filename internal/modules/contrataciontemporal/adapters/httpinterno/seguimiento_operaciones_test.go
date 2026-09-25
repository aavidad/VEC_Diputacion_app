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

type autoridadSeguimientoPrueba struct{ lecturaDenegada bool }

func (autoridadSeguimientoPrueba) ResolverContextoCanalSeguimiento(context.Context) (application.ContextoCanalSeguimiento, error) {
	return application.ContextoCanalSeguimiento{AutenticacionRef: "aut_" + strings.Repeat("a", 32), SesionRef: "ses_" + strings.Repeat("b", 32),
		PerfilRef: "prf_0123456789abcdefghijkl", OrganizacionRef: "organizacion:prueba"}, nil
}
func (a autoridadSeguimientoPrueba) AutorizarLecturaSeguimiento(context.Context, string, string) error {
	if a.lecturaDenegada {
		return ports.ErrAutorizacionDenegada
	}
	return nil
}

type ejecutorSeguimientoPrueba struct {
	cese  application.SolicitudRegistrarCese
	error error
}

func (e *ejecutorSeguimientoPrueba) recibo(op string) (ports.ReciboOperacionSeguimiento, error) {
	return ports.ReciboOperacionSeguimiento{Operacion: op, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		VersionAnterior: 7, VersionResultante: 8, FaseResultante: domain.FaseNombramiento, EstadoResultante: domain.EstadoEnCurso,
		ReciboRef: "recibo:prueba", AuditoriaRef: "aud_v3_" + strings.Repeat("0", 32), EventoRef: "evento:prueba", ActorRef: "per_prueba",
		RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), CausaClave: "fin_sustitucion", FechaEfecto: "2027-02-15"}, e.error
}
func (e *ejecutorSeguimientoPrueba) RegistrarCese(_ context.Context, s application.SolicitudRegistrarCese) (ports.ReciboOperacionSeguimiento, error) {
	e.cese = s
	return e.recibo(ports.OperacionRegistrarCese)
}
func (e *ejecutorSeguimientoPrueba) CerrarExpediente(context.Context, application.SolicitudCerrarExpediente) (ports.ReciboOperacionSeguimiento, error) {
	return e.recibo(ports.OperacionCerrarExpediente)
}
func (e *ejecutorSeguimientoPrueba) ModificarTrasNombramiento(context.Context, application.SolicitudModificarTrasNombramiento) (ports.ReciboOperacionSeguimiento, error) {
	return e.recibo(ports.OperacionModificarTrasNombramiento)
}
func (e *ejecutorSeguimientoPrueba) Opciones(context.Context) (ports.OpcionesSeguimiento, error) {
	return ports.OpcionesSeguimiento{Causas: []ports.CausaCese{{Clave: "fin_sustitucion", Etiqueta: "Fin de la sustitución", JustificanteTipo: "comunicacion_reincorporacion"}},
		Condiciones: []string{"cese_registrado"}, FaseRetorno: domain.FaseFiscalizacion}, nil
}
func (e *ejecutorSeguimientoPrueba) Estado(context.Context, string, string) (ports.EstadoSeguimientoExpediente, error) {
	return ports.EstadoSeguimientoExpediente{ExpedienteRef: "expediente:prueba"}, nil
}

func peticionSeguimiento(t *testing.T, manejadores map[string]http.Handler, ruta, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	manejadores[ruta].ServeHTTP(w, r)
	return w
}

func TestSeguimientoHTTPRegistraCeseYTraduceErrores(t *testing.T) {
	ejecutor := &ejecutorSeguimientoPrueba{}
	m, err := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":7,"clave_idempotencia":"11111111-1111-4111-8111-111111111111","causa_clave":"fin_sustitucion","fecha_efecto":"2027-02-15","justificante_ref":"documento:prueba","justificante_sha256":"` + strings.Repeat("a", 64) + `","observaciones":""}`
	w := peticionSeguimiento(t, m, RutaCesesNombramiento, cuerpo)
	if w.Code != http.StatusCreated || !ejecutor.cese.FechaEfecto.Equal(time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)) || ejecutor.cese.Canal.OrganizacionRef != "organizacion:prueba" {
		t.Fatalf("cese: %d %s", w.Code, w.Body.String())
	}
	var salida struct {
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Data["causa_clave"] != "fin_sustitucion" || salida.Data["recibo_ref"] != "recibo:prueba" {
		t.Fatalf("recibo: %s", w.Body.String())
	}
	if w := peticionSeguimiento(t, m, RutaCesesNombramiento, strings.Replace(cuerpo, "2027-02-15", "2027-02-30", 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("fecha imposible: %d", w.Code)
	}
	if w := peticionSeguimiento(t, m, RutaCesesNombramiento, strings.Replace(cuerpo, `"observaciones":""`, `"observaciones":"","actor_ref":"x"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("campo ajeno: %d", w.Code)
	}
	ejecutor.error = ports.ErrCeseSinIncorporacion
	w = peticionSeguimiento(t, m, RutaCesesNombramiento, cuerpo)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "sin_incorporacion") {
		t.Fatalf("sin incorporación: %d %s", w.Code, w.Body.String())
	}
	ejecutor.error = ports.ErrAutorizacionDenegada
	if w := peticionSeguimiento(t, m, RutaCierresExpediente, `{"expediente_ref":"expediente:prueba","version_esperada":8,"clave_idempotencia":"22222222-2222-4222-8222-222222222222","ginpix_numero":"","ginpix_confirmada_en":"","observaciones":""}`); w.Code != http.StatusForbidden {
		t.Fatalf("cierre denegado: %d", w.Code)
	}
}

func TestSeguimientoHTTPConsultaExigeLecturaAutorizada(t *testing.T) {
	ejecutor := &ejecutorSeguimientoPrueba{}
	m, _ := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, ejecutor)
	w := peticionSeguimiento(t, m, RutaSeguimientoCese, `{"expediente_ref":"expediente:prueba"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "comunicacion_reincorporacion") || !strings.Contains(w.Body.String(), `"cese":null`) {
		t.Fatalf("consulta: %d %s", w.Code, w.Body.String())
	}
	denegada, _ := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{lecturaDenegada: true}, ejecutor)
	if w := peticionSeguimiento(t, denegada, RutaSeguimientoCese, `{"expediente_ref":"expediente:prueba"}`); w.Code != http.StatusForbidden {
		t.Fatalf("lectura denegada: %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodGet, RutaSeguimientoCese, nil)
	w = httptest.NewRecorder()
	m[RutaSeguimientoCese].ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", w.Code)
	}
}
