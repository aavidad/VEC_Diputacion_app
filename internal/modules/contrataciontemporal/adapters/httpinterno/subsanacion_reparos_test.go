package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadSubsanacionReparosPrueba struct {
	contexto ContextoCanalSubsanacionReparos
}

func (a autoridadSubsanacionReparosPrueba) ResolverContextoCanalSubsanacionReparos(context.Context) (ContextoCanalSubsanacionReparos, error) {
	return a.contexto, nil
}

type ejecutorSubsanacionReparosPrueba struct {
	solicitud application.SolicitudRegistrarSubsanacionReparo
	err       error
	llamadas  int
}

func (e *ejecutorSubsanacionReparosPrueba) RegistrarSubsanacionReparo(_ context.Context, s application.SolicitudRegistrarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error) {
	e.solicitud = s
	e.llamadas++
	if e.err != nil {
		return ports.ReciboSubsanacionReparo{}, e.err
	}
	return ports.ReciboSubsanacionReparo{Operacion: ports.OperacionRegistrarSubsanacionReparo, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionAnterior: s.VersionEsperada, VersionResultante: s.VersionEsperada + 1, FaseResultante: domain.FaseSubsanacionUnidad, EstadoResultante: domain.EstadoIncidencia, ReciboRef: "recibo:subsanacion:http:001", AuditoriaRef: "auditoria:subsanacion:http:001", EventoRef: "evento:subsanacion:http:001", ActorRef: "actor:unidad:http:001", RegistradaEn: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)}, nil
}

func TestManejadorSubsanacionReparosRegistraYPublicaRecibo(t *testing.T) {
	e := &ejecutorSubsanacionReparosPrueba{}
	h, err := NuevoManejadorSubsanacionReparos(autoridadSubsanacionReparosPrueba{ContextoCanalSubsanacionReparos{"aut_aaaaaaaaaaaaaaaaaaaaaaaa", "ses_bbbbbbbbbbbbbbbbbbbbbbbb", "prf_cccccccccccccccccccccccc", "organizacion:subsanacion:http:001"}}, e)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaSubsanacionReparos, strings.NewReader(`{"expediente_ref":"expediente:subsanacion:http:001","version_esperada":6,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"Corrección sintética."}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || e.solicitud.Observaciones != "Corrección sintética." || e.solicitud.VersionEsperada != 6 {
		t.Fatalf("respuesta=%d solicitud=%#v", w.Code, e.solicitud)
	}
}

// Cubre el límite HTTP de la transición. La preparación SQL debe acreditar
// aparte que el conflicto se produce antes de confirmar otro efecto.
func TestManejadorSubsanacionHistoricaConPerfilNuevoDevuelve409SinRecibo(t *testing.T) {
	const perfilFijo = "prf_cccccccccccccccccccccccc"
	e := &ejecutorSubsanacionReparosPrueba{err: ports.ErrClaveIdempotenciaUsada}
	h, err := NuevoManejadorSubsanacionReparos(autoridadSubsanacionReparosPrueba{
		ContextoCanalSubsanacionReparos{"aut_aaaaaaaaaaaaaaaaaaaaaaaa", "ses_bbbbbbbbbbbbbbbbbbbbbbbb", perfilFijo, "organizacion:subsanacion:http:001"},
	}, e)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaSubsanacionReparos, strings.NewReader(`{"expediente_ref":"expediente:subsanacion:http:001","version_esperada":6,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"Corrección sintética."}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var respuesta envoltorioErrorCobertura
	if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusConflict || respuesta.Error.Codigo != "conflicto" ||
		e.llamadas != 1 || e.solicitud.PerfilRef != perfilFijo ||
		strings.Contains(w.Body.String(), "recibo_ref") || strings.Contains(w.Body.String(), "recibo:ct123:sub") {
		t.Fatalf("transición HTTP incorrecta: estado=%d respuesta=%s solicitud=%#v llamadas=%d", w.Code, w.Body.String(), e.solicitud, e.llamadas)
	}
}

// La proyección del detalle conserva la actuación histórica, pero su contrato
// público no incluye el recibo. La autorización de consulta se prueba en su
// propio caso de uso; esto no suplanta un GET de recibo de subsanación.
func TestProyeccionDetalleSubsanacionHistoricaConservaActuacionSinExponerRecibo(t *testing.T) {
	contenido, err := os.ReadFile(filepath.Join("..", "..", "domain", "testdata", "expediente_informe_nuevo_v8.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expediente domain.Expediente
	if err := json.Unmarshal(contenido, &expediente); err != nil {
		t.Fatal(err)
	}
	const reciboHistorico = "recibo:ct123:sub"
	if expediente.Validar() != nil || expediente.Version != 8 || len(expediente.Actuaciones) < 7 ||
		expediente.Actuaciones[6].ReciboRef != reciboHistorico ||
		expediente.Actuaciones[6].AccionClave != domain.AccionRegistrarSubsanacionReparo {
		t.Fatal("fixture histórica de subsanación inválida")
	}
	hitos := make([]ports.HitoExpedienteRRHH, len(expediente.Actuaciones))
	for indice, actuacion := range expediente.Actuaciones {
		hitos[indice] = ports.HitoExpedienteRRHH{
			Secuencia: actuacion.Secuencia, VersionExpediente: actuacion.VersionExpediente,
			AccionClave: actuacion.AccionClave, RealizadaEn: actuacion.RealizadaEn,
			FaseOrigen: actuacion.FaseOrigen, FaseDestino: actuacion.FaseDestino,
			EstadoOrigen: actuacion.EstadoOrigen, EstadoDestino: actuacion.EstadoDestino,
		}
	}
	proyeccion := proyectarDetalleRRHH(ports.DetalleExpedienteRRHH{Hitos: hitos})
	publicado, err := json.Marshal(proyeccion)
	if err != nil {
		t.Fatal(err)
	}
	if len(proyeccion.Hitos) != 8 || proyeccion.Hitos[6].AccionClave != string(domain.AccionRegistrarSubsanacionReparo) ||
		strings.Contains(string(publicado), reciboHistorico) || strings.Contains(string(publicado), "recibo_ref") {
		t.Fatalf("proyección histórica incorrecta: %s", publicado)
	}
}
