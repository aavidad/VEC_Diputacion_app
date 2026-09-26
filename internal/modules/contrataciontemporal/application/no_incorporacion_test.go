package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type noIncorporacionDoble struct {
	r       *reglasSeguimientoDoble
	segunda bool
	err     error
}

func (n noIncorporacionDoble) ReglaNoIncorporacion(context.Context, time.Time) (ports.ReglaNoIncorporacion, ports.PoliticaOperacionSeguimiento, error) {
	return ports.ReglaNoIncorporacion{SegundaPersona: n.segunda, Motivos: []ports.MotivoNoIncorporacion{
		{Clave: "no_presentado", Etiqueta: "No se presenta", ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo"}}}, n.r.politica("vec.contratacion_temporal.reglas"), n.err
}

func escenarioNoIncorporacion(t *testing.T, segunda bool) escenarioSeguimiento {
	t.Helper()
	e := nuevoEscenarioSeguimiento(t, 1)
	s, err := NuevoServicioOperacionesSeguimiento(DependenciasOperacionesSeguimiento{Contextos: e.servicio.contextos, Sellos: e.servicio.sellos,
		Repositorio: e.repo, Reglas: e.reglas, Autorizador: e.autorizador, Referencias: referenciasSeguimientoDoble{}, Coste: costeSeguimientoDoble{1},
		Lector: e.repo, Reloj: e.servicio.reloj, GINPIX: ginpixSeguimientoDoble{e.reglas}, Acreditada: &acreditadaDoble{},
		NoIncorporacion: noIncorporacionDoble{r: e.reglas, segunda: segunda}})
	if err != nil {
		t.Fatal(err)
	}
	e.servicio = s
	return e
}

func solicitudNoIncorporacionPrueba(e escenarioSeguimiento) SolicitudRegistrarNoIncorporacion {
	return SolicitudRegistrarNoIncorporacion{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444444", MotivoClave: "no_presentado", ResolucionRef: "resolucion:rrhh:2026/0142",
		ResolucionSHA256: strings.Repeat("b", 64), ResueltaPor: "per_segunda", FechaNotificacion: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}
}

func TestNoIncorporacionExigeLaIncorporacionAcreditada(t *testing.T) {
	e := nuevoEscenarioSeguimiento(t, 1)
	_, err := NuevoServicioOperacionesSeguimiento(DependenciasOperacionesSeguimiento{Contextos: e.servicio.contextos, Sellos: e.servicio.sellos,
		Repositorio: e.repo, Reglas: e.reglas, Autorizador: e.autorizador, Referencias: referenciasSeguimientoDoble{}, Coste: costeSeguimientoDoble{1},
		Lector: e.repo, Reloj: e.servicio.reloj, NoIncorporacion: noIncorporacionDoble{r: e.reglas}})
	if !errors.Is(err, ErrServicioSeguimientoInvalido) {
		t.Fatalf("no incorporación sin incorporación acreditada: %v", err)
	}
	if _, err := e.servicio.RegistrarNoIncorporacion(context.Background(), solicitudNoIncorporacionPrueba(e)); !errors.Is(err, ports.ErrOperacionSeguimientoNoDisponible) {
		t.Fatalf("sin composición la operación no existe: %v", err)
	}
}

func TestNoIncorporacionDevuelveAFiscalizacionConElCatalogo(t *testing.T) {
	e := escenarioNoIncorporacion(t, true)
	recibo, err := e.servicio.RegistrarNoIncorporacion(context.Background(), solicitudNoIncorporacionPrueba(e))
	if err != nil || recibo.VersionResultante != 8 || recibo.FaseResultante != domain.FaseFiscalizacion || recibo.EstadoResultante != domain.EstadoEnCurso {
		t.Fatalf("no incorporación: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	a := orden.Contexto.Atributos
	if orden.Operacion != ports.OperacionRegistrarNoIncorporacion || a["motivo_clave"] != "no_presentado" ||
		a["consecuencia_clave"] != "b24.sancion.baja_llamamiento_directo" || a["segunda_persona"] != "si" ||
		a["aceptacion_ref"] != "resolucion:aceptacion:prueba" || a["fecha_notificacion"] != "2026-09-20" ||
		e.autorizador.solicitudes[0].Audiencia != ports.AudienciaConsumoNoIncorporacionV1 ||
		e.autorizador.solicitudes[0].Recurso.Tipo != ports.TipoRecursoNoIncorporacion {
		t.Fatalf("contexto autorizado inesperado: %+v", a)
	}
	ultima := orden.Siguiente.Actuaciones[len(orden.Siguiente.Actuaciones)-1]
	if ultima.AccionClave != domain.AccionRegistrarNoIncorporacion || ultima.FaseOrigen != domain.FaseNombramiento ||
		ultima.FaseDestino != domain.FaseFiscalizacion || len(ultima.DocumentosRef) != 1 || ultima.DocumentosRef[0] != "resolucion:rrhh:2026/0142" {
		t.Fatalf("actuación: %+v", ultima)
	}
	if o, err := e.servicio.Opciones(context.Background()); err != nil || o.NoIncorporacion == nil || !o.NoIncorporacion.SegundaPersona ||
		len(o.NoIncorporacion.Motivos) != 1 {
		t.Fatalf("opciones con la no incorporación: %+v %v", o, err)
	}
}

func TestNoIncorporacionRechazaMotivoAjenoYMismaPersona(t *testing.T) {
	e := escenarioNoIncorporacion(t, true)
	sol := solicitudNoIncorporacionPrueba(e)
	sol.MotivoClave = "inventado"
	if _, err := e.servicio.RegistrarNoIncorporacion(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("motivo fuera del catálogo: %v", err)
	}
	// El actor autenticado de la escena es quien registra.
	otra := escenarioNoIncorporacion(t, true)
	if _, err := otra.servicio.RegistrarNoIncorporacion(context.Background(), solicitudNoIncorporacionPrueba(otra)); err != nil {
		t.Fatal(err)
	}
	actor := otra.repo.ordenes[0].Material.(ports.MaterialNoIncorporacion).ActorRef
	sol = solicitudNoIncorporacionPrueba(e)
	sol.ResueltaPor = actor
	if _, err := e.servicio.RegistrarNoIncorporacion(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("la segunda persona es quien registra: %v", err)
	}
	if len(e.autorizador.solicitudes) != 0 || e.repo.preparaciones != 0 {
		t.Fatal("una solicitud inválida llegó a autorizarse o prepararse")
	}
	sinSegunda := escenarioNoIncorporacion(t, false)
	sol = solicitudNoIncorporacionPrueba(sinSegunda)
	sol.ResueltaPor = actor
	if _, err := sinSegunda.servicio.RegistrarNoIncorporacion(context.Background(), sol); err != nil {
		t.Fatalf("sin segunda persona en el catálogo, la misma persona resuelve: %v", err)
	}
}
