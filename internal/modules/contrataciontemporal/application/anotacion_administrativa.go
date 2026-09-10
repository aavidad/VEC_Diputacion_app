package application

import (
	"context"
	"errors"
	"time"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const tiempoMaximoAnotacionAdministrativa = 15 * time.Second

var (
	ErrServicioAnotacionesAdministrativasInvalido  = errors.New("contratacion temporal: servicio de anotaciones administrativas invalido")
	ErrSolicitudAnotacionAdministrativaInvalida    = errors.New("contratacion temporal: solicitud de anotacion administrativa invalida")
	ErrAnotacionAdministrativaDenegada             = ports.ErrAutorizacionDenegada
	ErrResultadoAnotacionAdministrativaNoConfiable = errors.New("contratacion temporal: resultado de anotacion administrativa no confiable")
)

type SolicitudRegistrarAnotacionAdministrativa struct {
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	Observaciones     string
}

func (s SolicitudRegistrarAnotacionAdministrativa) Validar() error {
	if (ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef, PerfilRef: s.PerfilRef}).Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionEsperada == 0 || s.VersionEsperada > 9007199254740990 || !ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!textoAnotacionAdministrativaValido(s.Observaciones) {
		return ErrSolicitudAnotacionAdministrativaInvalida
	}
	return nil
}

type ServicioAnotacionesAdministrativas struct {
	recuperador   ports.RecuperadorMaterialAnotacionAdministrativaAutorizado
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	solicitudes   ports.ResolutorSolicitudPersonalAnotacionAdministrativa
	ambitos       ports.SelladorAmbitoAnotacionAdministrativa
	huellas       ports.DerivadorHuellaAnotacionAdministrativa
	preparaciones ports.PreparadorAnotacionAdministrativaIdempotente
	politicas     ports.ResolutorPoliticaAnotacionAdministrativa
	correlaciones vp.GeneradorReferenciasAutorizacionV2
	autorizador   vp.AutorizadorSolicitudLigadaV3
	reloj         ports.Reloj
	transaccion   ports.TransaccionAnotacionesAdministrativas
}

func NuevoServicioAnotacionesAdministrativas(contextos ports.ResolutorContextoAutorizacionAltaV3, solicitudes ports.ResolutorSolicitudPersonalAnotacionAdministrativa, ambitos ports.SelladorAmbitoAnotacionAdministrativa, huellas ports.DerivadorHuellaAnotacionAdministrativa, preparaciones ports.PreparadorAnotacionAdministrativaIdempotente, politicas ports.ResolutorPoliticaAnotacionAdministrativa, correlaciones vp.GeneradorReferenciasAutorizacionV2, autorizador vp.AutorizadorSolicitudLigadaV3, reloj ports.Reloj, transaccion ports.TransaccionAnotacionesAdministrativas) (*ServicioAnotacionesAdministrativas, error) {
	for _, d := range []any{contextos, solicitudes, ambitos, huellas, preparaciones, politicas, correlaciones, autorizador, reloj, transaccion} {
		if dependenciaNula(d) {
			return nil, ErrServicioAnotacionesAdministrativasInvalido
		}
	}
	return &ServicioAnotacionesAdministrativas{nil, contextos, solicitudes, ambitos, huellas, preparaciones, politicas, correlaciones, autorizador, reloj, transaccion}, nil
}

func (s *ServicioAnotacionesAdministrativas) Registrar(ctx context.Context, solicitud SolicitudRegistrarAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error) {
	return s.registrar(ctx, solicitud, false)
}
func (s *ServicioAnotacionesAdministrativas) registrar(ctx context.Context, solicitud SolicitudRegistrarAnotacionAdministrativa, soloRecuperacion bool) (ports.ReciboAnotacionAdministrativa, error) {
	var cero ports.ReciboAnotacionAdministrativa
	if s == nil || ctx == nil || solicitud.Validar() != nil {
		return cero, ErrSolicitudAnotacionAdministrativaInvalida
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	op, cancelar := context.WithTimeout(ctx, tiempoMaximoAnotacionAdministrativa)
	defer cancelar()
	contextoSolicitud := ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: solicitud.AutenticacionRef, SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(op, contextoSolicitud)
	ahora := instanteCanonico(s.reloj.Ahora())
	if err != nil || contexto.ValidarPara(contextoSolicitud, ahora) != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	actor, err := contexto.Vinculo.Datos()
	if err != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	solicitudPersonal, err := s.solicitudes.ResolverSolicitudPersonalAnotacionAdministrativa(op, solicitud.OrganizacionRef, solicitud.ExpedienteRef)
	if err != nil || !domain.ReferenciaOpacaValida(solicitudPersonal) {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	material := ports.MaterialAnotacionAdministrativa{OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef, SolicitudPersonalRef: solicitudPersonal, VersionEsperada: solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia, Observaciones: solicitud.Observaciones, ActorRef: actor.PrincipalID, PerfilRef: actor.PerfilActivoRef}

	ambitos, err := s.ambitos.SellarAmbitoAnotacionAdministrativa(op, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: material.ClaveIdempotencia, OrganizacionRef: material.OrganizacionRef, ActorRef: material.ActorRef, PerfilRef: material.PerfilRef})
	if err != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, err)
	}
	huellas, err := s.huellas.DerivarHuellaAnotacionAdministrativa(op, material)
	if err != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, err)
	}
	preparar := ports.SolicitudPrepararAnotacionAdministrativa{Material: material, AmbitosHMAC: ambitos, HuellasPeticionHMAC: huellas}
	if preparar.Validar() != nil {
		return cero, ErrResultadoAnotacionAdministrativaNoConfiable
	}
	preparacion, err := s.preparaciones.PrepararAnotacionAdministrativa(op, preparar)
	if err != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, err)
	}
	if preparacion.ValidarPara(preparar) != nil {
		return cero, ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if soloRecuperacion && preparacion.Estado != ports.PreparacionAnotacionAdministrativaConfirmada {
		return cero, ErrResultadoAnotacionAdministrativaNoConfiable
	}
	// Nunca se devuelve un recibo obtenido por preparación sin V3 actual.
	politicaSolicitud := ports.SolicitudResolverPoliticaAnotacionAdministrativa{Material: material, Preparacion: preparacion, Instante: instanteCanonico(s.reloj.Ahora())}
	politica, err := s.politicas.ResolverPoliticaAnotacionAdministrativa(op, politicaSolicitud)
	if err != nil || politica.ValidarPara(politicaSolicitud, politicaSolicitud.Instante) != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(op, s.correlaciones)
	if err != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	solicitudV3, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: contexto.Vinculo, ReferenciaMotivo: politica.MotivoAutorizacion, Accion: string(domain.AccionRegistrarAnotacionAdministrativa), Recurso: ports.RecursoAutorizacionAnotacionAdministrativa(preparacion, politica), Finalidad: ports.FinalidadRegistrarAnotacionAdministrativa, Correlacion: correlacion})
	if err != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(op, solicitudV3, contexto.Resultado)
	efecto := instanteCanonico(s.reloj.Ahora())
	if err != nil || contexto.ValidarPara(contextoSolicitud, efecto) != nil || politica.ValidarPara(politicaSolicitud, efecto) != nil || !autorizacionV3ValidaEn(solicitudV3, decision, confirmacion, efecto) {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	var siguiente domain.Expediente
	if preparacion.Estado == ports.PreparacionAnotacionAdministrativaPreparada {
		siguiente, err = preparacion.Expediente.RegistrarAnotacionAdministrativa(material.VersionEsperada, preparacion.SeguimientoOriginal, domain.DatosActuacion{AccionClave: domain.AccionRegistrarAnotacionAdministrativa, ActorRef: material.ActorRef, UnidadRef: preparacion.Expediente.Asignacion.UnidadRef, ReciboRef: preparacion.Referencias.ReciboRef, RealizadaEn: efecto, FaseDestino: preparacion.Expediente.FaseActual, EstadoDestino: preparacion.Expediente.EstadoActual, Observaciones: material.Observaciones})
		if err != nil {
			return cero, ErrResultadoAnotacionAdministrativaNoConfiable
		}
	}
	orden := ports.OrdenConfirmarAnotacionAdministrativa{Material: material, ExpedienteSiguiente: siguiente, SeguimientoOriginal: preparacion.SeguimientoOriginal, Referencias: preparacion.Referencias, InstanteEfecto: efecto, Preparacion: preparacion, Politica: politica, Evidencia: ports.EvidenciaAutorizacionAnotacionAdministrativa{Contexto: contexto, SolicitudV3: solicitudV3, DecisionV3: decision, ConfirmacionV3: confirmacion}}
	recibo, err := s.transaccion.ConfirmarAnotacionAdministrativa(op, orden)
	if err != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, err)
	}
	if recibo.ValidarParaPreparacion(preparacion) != nil {
		return cero, ErrResultadoAnotacionAdministrativaNoConfiable
	}

	return recibo, nil
}

func (s *ServicioAnotacionesAdministrativas) RegistrarAnotacionAdministrativa(ctx context.Context, solicitud SolicitudRegistrarAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error) {
	return s.Registrar(ctx, solicitud)
}

func textoAnotacionAdministrativaValido(v string) bool {
	return ports.ValidarResultadoFiscalizacion(domain.FiscalizacionFavorableConObservaciones, v) == nil
}
func clasificarFalloAnotacionAdministrativa(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, domain.ErrVersionEnConflicto) {
		return domain.ErrVersionEnConflicto
	}
	if errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		return ports.ErrClaveIdempotenciaUsada
	}
	if errors.Is(err, ports.ErrAutorizacionDenegada) {
		return ErrAnotacionAdministrativaDenegada
	}
	if errors.Is(err, ports.ErrResultadoAnotacionAdministrativaNoConfiable) {
		return ErrResultadoAnotacionAdministrativaNoConfiable
	}
	return ports.ErrPersistenciaAnotacionAdministrativaNoDisponible
}

// ConRecuperacionAutorizada devuelve otra composición; nunca activa una fuente
// que sólo busca por clave o HMAC sin permiso de lectura nominal actual.
func (s *ServicioAnotacionesAdministrativas) ConRecuperacionAutorizada(r ports.RecuperadorMaterialAnotacionAdministrativaAutorizado) (*ServicioAnotacionesAdministrativas, error) {
	if s == nil || dependenciaNula(r) {
		return nil, ErrServicioAnotacionesAdministrativasInvalido
	}
	c := *s
	c.recuperador = r
	return &c, nil
}
func (s *ServicioAnotacionesAdministrativas) RecuperarAnotacionAdministrativa(ctx context.Context, solicitud SolicitudRegistrarAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error) {
	var cero ports.ReciboAnotacionAdministrativa
	if s == nil || ctx == nil || dependenciaNula(s.recuperador) {
		return cero, ports.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	if solicitud.Observaciones != "" || solicitud.VersionEsperada != 0 || !domain.ReferenciaOpacaValida(solicitud.OrganizacionRef) || !domain.ReferenciaOpacaValida(solicitud.ExpedienteRef) || !ports.ClaveIdempotenciaValida(solicitud.ClaveIdempotencia) {
		return cero, ErrSolicitudAnotacionAdministrativaInvalida
	}
	op, cancel := context.WithTimeout(ctx, tiempoMaximoAnotacionAdministrativa)
	defer cancel()
	sc := ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: solicitud.AutenticacionRef, SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef}
	c, e := s.contextos.ResolverContextoAutorizacionAltaV3(op, sc)
	if e != nil || c.ValidarPara(sc, instanteCanonico(s.reloj.Ahora())) != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	v, e := c.Vinculo.Datos()
	if e != nil {
		return cero, ErrAnotacionAdministrativaDenegada
	}
	ambitos, e := s.ambitos.SellarAmbitoAnotacionAdministrativa(op, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: solicitud.ClaveIdempotencia, OrganizacionRef: solicitud.OrganizacionRef, ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef})
	if e != nil || ambitos.ValidarDominio(ports.DominioAmbitoIdempotenciaAnotacionAdministrativa) != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, e)
	}
	m, e := s.recuperador.RecuperarMaterialAnotacionAdministrativaAutorizada(op, ports.SolicitudRecuperarMaterialAnotacionAdministrativa{Contexto: c, OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef, ClaveIdempotencia: solicitud.ClaveIdempotencia, ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, AmbitosHMAC: ambitos})
	if e != nil {
		return cero, clasificarFalloAnotacionAdministrativa(op, e)
	}
	if m.Validar() != nil || m.OrganizacionRef != solicitud.OrganizacionRef || m.ExpedienteRef != solicitud.ExpedienteRef || m.ClaveIdempotencia != solicitud.ClaveIdempotencia || m.ActorRef != v.PrincipalID || m.PerfilRef != v.PerfilActivoRef {
		return cero, ErrResultadoAnotacionAdministrativaNoConfiable
	}
	solicitud.VersionEsperada = m.VersionEsperada
	solicitud.Observaciones = m.Observaciones
	return s.registrar(op, solicitud, true)
}
