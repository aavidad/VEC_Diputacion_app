package bootstrap

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestConsultaCircuitoAuditaDenegacionFinalFueraDeLaLectura(t *testing.T) {
	registrador := &auditorConsultaReciboPrueba{}
	estado := http.StatusNotFound
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(estado)
		_, _ = w.Write([]byte(`{"error":{"correlacion_ref":"corr_no_disponible"}}`))
	})
	auditor := auditorConsultaCircuitoRRHHDenegada{
		siguiente: siguiente, registrador: registrador,
		reloj: relojContratacionTemporalDesarrollo{},
	}
	peticion := httptest.NewRequest(http.MethodPost, httpinterno.RutaConsultaCircuitoRRHH, nil)
	respuesta := httptest.NewRecorder()
	auditor.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusNotFound || len(registrador.ordenes) != 1 ||
		registrador.ordenes[0].Ruta != httpinterno.RutaConsultaCircuitoRRHH ||
		registrador.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		t.Fatal("la denegación final no quedó registrada antes de responder")
	}
	registrador.err = errors.New("bitacora no disponible")
	respuesta = httptest.NewRecorder()
	auditor.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatal("la denegación salió sin bitácora")
	}
	registrador.err = nil
	estado = http.StatusOK
	respuesta = httptest.NewRecorder()
	auditor.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || len(registrador.ordenes) != 2 {
		t.Fatal("la lectura concedida añadió una denegación")
	}
}

func TestConsultaCircuitoRRHHV3LigaSolicitudYRecursoExactos(t *testing.T) {
	s := ports.SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:    "expediente:ct:circuito:001",
		VersionObservada: 3,
	}
	recurso, err := postgresct.RecursoConsultaCircuitoRRHH(s)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), claveConsultaCircuitoRRHHDesarrollo{}, s)
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Accion: postgresct.AccionConsultaCircuitoRRHH, Recurso: recurso,
		Finalidad:        "gestionar_contratacion_temporal",
		ReferenciaMotivo: motivoConsultaCircuitoRRHHDesarrollo(),
	}
	if !solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("la consulta exacta se ha denegado")
	}
	datos.Recurso.Atributos = maps.Clone(recurso.Atributos)
	datos.Recurso.Atributos["material_sha256"] = huellaAltaContratacionTemporalDesarrollo("otra-consulta")
	if solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("se aceptó otra versión o expediente con el mismo permiso")
	}
	datos.Recurso = recurso
	datos.Accion = ports.AccionConsultarDetalleRRHH
	if solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("se aceptó un permiso de consulta del detalle antiguo")
	}
}

func TestConsultaCircuitoRRHHResuelveMotivoNominal(t *testing.T) {
	soporte := &soporteAltaContratacionTemporalDesarrollo{}
	motivo, ok := soporte.motivoAutorizacionParaRuta(httpinterno.RutaConsultaCircuitoRRHH)
	if !ok || motivo != motivoConsultaCircuitoRRHHDesarrollo() {
		t.Fatal("la ruta de consulta debe resolver su motivo nominal")
	}
}
