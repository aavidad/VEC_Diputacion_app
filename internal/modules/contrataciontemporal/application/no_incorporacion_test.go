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

// solicitudNoIncorporacionPrueba: por defecto, la propuesta de cuatro ojos.
func solicitudNoIncorporacionPrueba(e escenarioSeguimiento) SolicitudRegistrarNoIncorporacion {
	return SolicitudRegistrarNoIncorporacion{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444444", Paso: domain.PasoNoIncorporacionProponer, MotivoClave: "no_presentado",
		ResolucionRef: "resolucion:rrhh:2026/0142", ResolucionSHA256: strings.Repeat("b", 64), FechaNotificacion: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}
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

// Con segunda persona, proponer no tiene efecto y confirmar devuelve a
// fiscalización; quien resuelve es siempre el actor autenticado que confirma.
func TestNoIncorporacionCuatroOjosConElCatalogo(t *testing.T) {
	e := escenarioNoIncorporacion(t, true)
	recibo, err := e.servicio.RegistrarNoIncorporacion(context.Background(), solicitudNoIncorporacionPrueba(e))
	if err != nil || recibo.VersionResultante != 8 || recibo.FaseResultante != domain.FaseNombramiento {
		t.Fatalf("propuesta: %+v %v", recibo, err)
	}
	propuesta := e.repo.ordenes[0]
	if a := propuesta.Contexto.Atributos; a["paso"] != domain.PasoNoIncorporacionProponer || a["propuesta_ref"] != "" || a["resuelta_por"] != "" {
		t.Fatalf("contexto de la propuesta: %+v", a)
	}
	if ultima := propuesta.Siguiente.Actuaciones[len(propuesta.Siguiente.Actuaciones)-1]; ultima.AccionClave != domain.AccionProponerNoIncorporacion ||
		ultima.FaseDestino != domain.FaseNombramiento {
		t.Fatalf("actuación de la propuesta: %+v", ultima)
	}
	e = escenarioNoIncorporacion(t, true)
	sol := solicitudNoIncorporacionPrueba(e)
	sol.Paso, sol.PropuestaRef = domain.PasoNoIncorporacionConfirmar, "recibo:ct124:np1"
	recibo, err = e.servicio.RegistrarNoIncorporacion(context.Background(), sol)
	if err != nil || recibo.VersionResultante != 8 || recibo.FaseResultante != domain.FaseFiscalizacion || recibo.EstadoResultante != domain.EstadoEnCurso {
		t.Fatalf("confirmación: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	actor := orden.Material.(ports.MaterialNoIncorporacion).ActorRef
	a := orden.Contexto.Atributos
	if orden.Operacion != ports.OperacionRegistrarNoIncorporacion || a["paso"] != domain.PasoNoIncorporacionConfirmar ||
		a["propuesta_ref"] != "recibo:ct124:np1" || a["resuelta_por"] != actor || a["motivo_clave"] != "no_presentado" ||
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
	e = escenarioNoIncorporacion(t, true)
	sol.Canal, sol.Paso = e.canal, domain.PasoNoIncorporacionRechazar
	recibo, err = e.servicio.RegistrarNoIncorporacion(context.Background(), sol)
	if err != nil || recibo.FaseResultante != domain.FaseNombramiento ||
		e.repo.ordenes[0].Siguiente.Actuaciones[len(e.repo.ordenes[0].Siguiente.Actuaciones)-1].AccionClave != domain.AccionRechazarNoIncorporacion {
		t.Fatalf("rechazo: %+v %v", recibo, err)
	}
}

func TestNoIncorporacionRechazaPasosIncoherentesConElCatalogo(t *testing.T) {
	e := escenarioNoIncorporacion(t, true)
	invalidas := map[string]func(*SolicitudRegistrarNoIncorporacion){
		"motivo ajeno":                func(s *SolicitudRegistrarNoIncorporacion) { s.MotivoClave = "inventado" },
		"propuesta que declara quien": func(s *SolicitudRegistrarNoIncorporacion) { s.ResueltaPor = "per_otra" },
		"registro de un paso": func(s *SolicitudRegistrarNoIncorporacion) {
			s.Paso, s.ResueltaPor = domain.PasoNoIncorporacionRegistrar, "per_otra"
		},
		"confirmación sin propuesta": func(s *SolicitudRegistrarNoIncorporacion) { s.Paso = domain.PasoNoIncorporacionConfirmar },
		"confirmación que declara quien": func(s *SolicitudRegistrarNoIncorporacion) {
			s.Paso, s.PropuestaRef, s.ResueltaPor = domain.PasoNoIncorporacionConfirmar, "recibo:ct124:np1", "per_otra"
		},
		"paso desconocido": func(s *SolicitudRegistrarNoIncorporacion) { s.Paso = "firmar" },
	}
	for nombre, mutar := range invalidas {
		sol := solicitudNoIncorporacionPrueba(e)
		mutar(&sol)
		if _, err := e.servicio.RegistrarNoIncorporacion(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
	if len(e.autorizador.solicitudes) != 0 || e.repo.preparaciones != 0 {
		t.Fatal("una solicitud inválida llegó a autorizarse o prepararse")
	}
	// Sin segunda persona: un solo paso, con quien resolvió declarado.
	sinSegunda := escenarioNoIncorporacion(t, false)
	sol := solicitudNoIncorporacionPrueba(sinSegunda)
	if _, err := sinSegunda.servicio.RegistrarNoIncorporacion(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("sin segunda persona no hay propuesta: %v", err)
	}
	sol.Paso, sol.ResueltaPor = domain.PasoNoIncorporacionRegistrar, "per_declarada"
	recibo, err := sinSegunda.servicio.RegistrarNoIncorporacion(context.Background(), sol)
	if err != nil || recibo.FaseResultante != domain.FaseFiscalizacion {
		t.Fatalf("sin segunda persona en el catálogo se registra en un paso: %+v %v", recibo, err)
	}
}
