package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrServicioCancelacionInvalido = errors.New("contratacion temporal: servicio de cancelacion invalido")

// SolicitudCancelarExpediente es lo que aporta quien cancela. Actor, perfil y
// organización proceden del canal autenticado, nunca del cuerpo.
type SolicitudCancelarExpediente struct {
	Canal             ContextoCanalSeguimiento
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	MotivoClave       domain.ClaveCatalogo
	Observaciones     string
}

// OpcionesCancelacion son las fases y los motivos vigentes para el canal.
// Todos conserva los motivos de cualquier canal para nombrar una
// cancelación ya registrada por otro.
type OpcionesCancelacion struct {
	Fases   []domain.ClaveFase
	Motivos []ports.MotivoCancelacion
	Todos   []ports.MotivoCancelacion
}

// ServicioCancelacionExpediente coordina la cancelación de un canal (RRHH o
// centro). Comparte con las operaciones de seguimiento la identidad, los
// sellos, la preparación y la confirmación; las fases y los motivos proceden
// siempre de FuenteReglasCancelacion.
type ServicioCancelacionExpediente struct {
	canal  domain.CanalCancelacion
	nucleo *ServicioOperacionesSeguimiento
	reglas ports.FuenteReglasCancelacion
	lector ports.LectorEstadoCancelacion
}

type DependenciasCancelacionExpediente struct {
	Canal       domain.CanalCancelacion
	Contextos   ports.ResolutorContextoAutorizacionAltaV3
	Sellos      ports.SelladorOperacionSeguimiento
	Repositorio ports.RepositorioOperacionSeguimiento
	Reglas      ports.FuenteReglasCancelacion
	Autorizador ports.AutorizadorOperacionSeguimiento
	Referencias ports.GeneradorReferenciasSeguimiento
	Lector      ports.LectorEstadoCancelacion
	Reloj       ports.Reloj
}

func NuevoServicioCancelacionExpediente(d DependenciasCancelacionExpediente) (*ServicioCancelacionExpediente, error) {
	if !d.Canal.Valido() || dependenciaNula(d.Contextos) || dependenciaNula(d.Sellos) || dependenciaNula(d.Repositorio) ||
		dependenciaNula(d.Reglas) || dependenciaNula(d.Autorizador) || dependenciaNula(d.Referencias) || dependenciaNula(d.Lector) ||
		dependenciaNula(d.Reloj) {
		return nil, ErrServicioCancelacionInvalido
	}
	nucleo := &ServicioOperacionesSeguimiento{contextos: d.Contextos, sellos: d.Sellos, repositorio: d.Repositorio,
		autorizador: d.Autorizador, referencias: d.Referencias, reloj: d.Reloj}
	return &ServicioCancelacionExpediente{canal: d.Canal, nucleo: nucleo, reglas: d.Reglas, lector: d.Lector}, nil
}

func (s *ServicioCancelacionExpediente) regla(ctx context.Context) (ports.ReglaCancelacion, ports.PoliticaOperacionSeguimiento, error) {
	ahora := instanteCanonico(s.nucleo.reloj.Ahora())
	regla, politica, err := s.reglas.ReglaCancelacion(ctx, ahora)
	if err != nil || !regla.Valida() || !politica.ValidaEn(ahora) {
		return ports.ReglaCancelacion{}, ports.PoliticaOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	return regla, politica, nil
}

func textoFases(fases []domain.ClaveFase) string {
	partes := make([]string, len(fases))
	for i, f := range fases {
		partes[i] = string(f)
	}
	return strings.Join(partes, ",")
}

// CancelarExpediente valida el motivo contra el catálogo vigente para el
// canal. Fase, estado, fiscalización y versión los comprueban dominio y SQL.
func (s *ServicioCancelacionExpediente) CancelarExpediente(ctx context.Context, sol SolicitudCancelarExpediente) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil || s.nucleo == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioCancelacionInvalido
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.nucleo.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	regla, politica, err := s.regla(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	admitido := false
	for _, m := range regla.Motivos {
		admitido = admitido || (m.Clave == sol.MotivoClave && m.AdmiteCanal(s.canal))
	}
	if !admitido {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	datos := domain.DatosCancelacion{MotivoClave: sol.MotivoClave, Observaciones: sol.Observaciones, Canal: s.canal,
		FasesAdmitidas: append([]domain.ClaveFase(nil), regla.Fases...)}
	material := ports.MaterialCancelacion{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor,
		PerfilRef: perfil, VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	fases := textoFases(datos.FasesAdmitidas)
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Canal, Motivo, Fases, Observaciones string
		Version                                                                                 uint64
	}{ports.OperacionCancelarExpediente, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, string(s.canal),
		string(datos.MotivoClave), fases, datos.Observaciones, sol.VersionEsperada})
	prep, err := s.nucleo.preparar(ctx, ports.OperacionCancelarExpediente, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	// La cancelación no cambia de fase: la fase previa es la del expediente
	// leído (vigente o, en la repetición, el resultado ya confirmado).
	if prep.Expediente.Referencia != sol.ExpedienteRef || prep.Expediente.OrganizacionRef != material.OrganizacionRef ||
		!prep.Expediente.FaseActual.Valida() {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
	}
	instante := instanteCanonico(s.nucleo.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionCancelarExpediente, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionCancelarExpediente, Finalidad: ports.FinalidadCancelarExpediente,
		Audiencia: ports.AudienciaConsumoCancelacionV1}
	ambitos := map[string]string{"organizacion_ref": material.OrganizacionRef, "expediente_ref": material.ExpedienteRef,
		"fase_previa": string(prep.Expediente.FaseActual), "estado_previo": string(domain.EstadoEnCurso)}
	if s.canal == domain.CanalCancelacionCentro {
		ambitos["centro_ref"] = prep.Expediente.Solicitud.CentroRef
	}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitos,
		Atributos: atributosPoliticaSeguimiento(map[string]string{"canal": string(s.canal), "motivo_clave": string(datos.MotivoClave),
			"fases_admitidas": fases, "observaciones_huella_sha256": huellaTexto(datos.Observaciones)}, politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		if prep.Expediente.Version != sol.VersionEsperada {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		orden.Siguiente, err = prep.Expediente.Cancelar(sol.VersionEsperada, datos, domain.DatosActuacion{AccionClave: domain.AccionCancelarExpediente,
			ActorRef: actor, UnidadRef: prep.Expediente.UnidadActual(), ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: prep.Expediente.FaseActual, EstadoDestino: domain.EstadoCancelado, Observaciones: datos.Observaciones})
		if err != nil {
			if prep.Expediente.Fiscalizado() {
				return ports.ReciboOperacionSeguimiento{}, ports.ErrCancelacionTrasFiscalizacion
			}
			if !prep.Expediente.CancelableEn(datos.FasesAdmitidas) {
				return ports.ReciboOperacionSeguimiento{}, ports.ErrCancelacionNoAdmitida
			}
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.nucleo.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}

// Opciones publica las fases y los motivos vigentes que puede usar el canal.
func (s *ServicioCancelacionExpediente) Opciones(ctx context.Context) (OpcionesCancelacion, error) {
	if s == nil || s.nucleo == nil || ctx == nil {
		return OpcionesCancelacion{}, ErrServicioCancelacionInvalido
	}
	regla, _, err := s.regla(ctx)
	if err != nil {
		return OpcionesCancelacion{}, err
	}
	o := OpcionesCancelacion{Fases: append([]domain.ClaveFase(nil), regla.Fases...), Todos: append([]ports.MotivoCancelacion(nil), regla.Motivos...)}
	for _, m := range regla.Motivos {
		if m.AdmiteCanal(s.canal) {
			o.Motivos = append(o.Motivos, m)
		}
	}
	if len(o.Motivos) == 0 {
		return OpcionesCancelacion{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	return o, nil
}

// Estado devuelve la cancelación registrada. La composición solo lo invoca
// tras acreditar la lectura del expediente exacto.
func (s *ServicioCancelacionExpediente) Estado(ctx context.Context, organizacionRef, expedienteRef string) (ports.EstadoCancelacionExpediente, error) {
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.EstadoCancelacionExpediente{}, ErrSolicitudSeguimientoInvalida
	}
	return s.lector.ConsultarEstadoCancelacion(ctx, organizacionRef, expedienteRef)
}
