package application

import (
	"context"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// SolicitudRegistrarNoIncorporacion es lo que RRHH envía: el motivo del
// catálogo, la resolución (referencia y huella), quién la resolvió y cuándo
// se notificó. La consecuencia y la segregación las fija el catálogo.
type SolicitudRegistrarNoIncorporacion struct {
	Canal             ContextoCanalSeguimiento
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	MotivoClave       string
	ResolucionRef     string
	ResolucionSHA256  string
	ResueltaPor       string
	FechaNotificacion time.Time
	Observaciones     string
}

// RegistrarNoIncorporacion registra la no incorporación con la regla c22
// vigente. La aceptación, la ausencia de incorporación y la unicidad las
// comprueba SQL; la baja la aplica Bolsa al recibir el evento.
func (s *ServicioOperacionesSeguimiento) RegistrarNoIncorporacion(ctx context.Context, sol SolicitudRegistrarNoIncorporacion) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	if s.noIncorporacion == nil {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	regla, politica, err := s.noIncorporacion.ReglaNoIncorporacion(ctx, ahora)
	if err != nil || !regla.Valida() || !politica.ValidaEn(ahora) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	motivo, admitido := regla.Motivo(sol.MotivoClave)
	if !admitido {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	datos := domain.DatosNoIncorporacion{MotivoClave: motivo.Clave, ConsecuenciaClave: motivo.ConsecuenciaClave,
		ResolucionRef: sol.ResolucionRef, ResolucionSHA256: sol.ResolucionSHA256, ResueltaPor: sol.ResueltaPor,
		SegundaPersona: regla.SegundaPersona, FechaNotificacion: sol.FechaNotificacion, Observaciones: sol.Observaciones}
	material := ports.MaterialNoIncorporacion{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor,
		PerfilRef: perfil, VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	fecha := datos.FechaNotificacion.Format(time.DateOnly)
	segunda := "no"
	if datos.SegundaPersona {
		segunda = "si"
	}
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Motivo, Consecuencia, Resolucion, ResolucionSHA256, ResueltaPor, Segunda, Fecha, Observaciones string
		Version                                                                                                                                            uint64
	}{ports.OperacionRegistrarNoIncorporacion, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, datos.MotivoClave,
		datos.ConsecuenciaClave, datos.ResolucionRef, datos.ResolucionSHA256, datos.ResueltaPor, segunda, fecha, datos.Observaciones, sol.VersionEsperada})
	prep, err := s.preparar(ctx, ports.OperacionRegistrarNoIncorporacion, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionRegistrarNoIncorporacion, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionRegistrarNoIncorporacion, Finalidad: ports.FinalidadRegistrarNoIncorporacion,
		Audiencia: ports.AudienciaConsumoNoIncorporacionV1}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosSeguimiento(material.OrganizacionRef, material.ExpedienteRef),
		Atributos: atributosPoliticaSeguimiento(map[string]string{"motivo_clave": datos.MotivoClave, "consecuencia_clave": datos.ConsecuenciaClave,
			"resolucion_ref": datos.ResolucionRef, "resolucion_sha256": datos.ResolucionSHA256, "resuelta_por": datos.ResueltaPor,
			"segunda_persona": segunda, "fecha_notificacion": fecha, "observaciones_huella_sha256": huellaTexto(datos.Observaciones),
			"aceptacion_ref": prep.AceptacionRef}, politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		if !domain.ReferenciaOpacaValida(prep.AceptacionRef) || prep.Expediente.Version != sol.VersionEsperada ||
			prep.Expediente.Referencia != sol.ExpedienteRef || prep.Expediente.OrganizacionRef != material.OrganizacionRef ||
			prep.Expediente.Asignacion == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		orden.Siguiente, err = prep.Expediente.RegistrarNoIncorporacion(sol.VersionEsperada, datos, domain.DatosActuacion{
			AccionClave: domain.AccionRegistrarNoIncorporacion, ActorRef: actor, UnidadRef: prep.Expediente.Asignacion.UnidadRef,
			ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante, FaseDestino: domain.FaseFiscalizacion,
			EstadoDestino: domain.EstadoEnCurso, Observaciones: datos.Observaciones, DocumentosRef: []string{datos.ResolucionRef}})
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}
