package contactopropio

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/usuarios"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type DependenciasOperaciones struct {
	EmisorPreparar ports.AutorizadorContactoUsuario
	EmisorCancelar ports.AutorizadorContactoUsuario
	EmisorListar   ports.AutorizadorContactoUsuario
	EmisorDetalle  ports.AutorizadorContactoUsuario
	MotivoPreparar domain.ReferenciaEntradaCatalogo
	MotivoCancelar domain.ReferenciaEntradaCatalogo
	MotivoListar   domain.ReferenciaEntradaCatalogo
	MotivoDetalle  domain.ReferenciaEntradaCatalogo
}

type ServicioOperaciones struct {
	*Servicio
	repositorio             ports.RepositorioOperacionesContactoUsuario
	dependenciasOperaciones DependenciasOperaciones
}

var _ EjecutorOperacionesContactoPropio = (*ServicioOperaciones)(nil)

type emisorOperacionesContacto struct{ d DependenciasOperaciones }

func (e emisorOperacionesContacto) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud domain.SolicitudAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrContactoPropioNoDisponible
	}
	var emisor ports.AutorizadorContactoUsuario
	switch datos.Accion {
	case application.AccionPrepararOperacionContacto:
		emisor = e.d.EmisorPreparar
	case application.AccionCancelarOperacionContacto:
		emisor = e.d.EmisorCancelar
	case application.AccionListarOperacionesContacto:
		emisor = e.d.EmisorListar
	case application.AccionDetalleOperacionContacto:
		emisor = e.d.EmisorDetalle
	default:
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrContactoPropioNoDisponible
	}
	if dependenciaContactoPropioNula(emisor) {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrContactoPropioNoDisponible
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
}

type generadorOperacionContacto struct{}

func (generadorOperacionContacto) NuevaOperacionContactoUsuario(ctx context.Context) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", ErrContactoPropioNoDisponible
	}
	var aleatorio [24]byte
	if _, err := rand.Read(aleatorio[:]); err != nil || ctx.Err() != nil {
		return "", ErrContactoPropioNoDisponible
	}
	return "opr_" + base64.RawURLEncoding.EncodeToString(aleatorio[:]), nil
}

func NuevoServicioOperaciones(s *Servicio, d DependenciasOperaciones) (*ServicioOperaciones, error) {
	if s == nil || s.dependencias.PoolEscritor == nil || dependenciaContactoPropioNula(s.registro) {
		return nil, ErrContactoPropioNoDisponible
	}
	for _, x := range []ports.AutorizadorContactoUsuario{d.EmisorPreparar, d.EmisorCancelar, d.EmisorListar, d.EmisorDetalle} {
		if dependenciaContactoPropioNula(x) {
			return nil, ErrContactoPropioNoDisponible
		}
	}
	for _, motivo := range []domain.ReferenciaEntradaCatalogo{d.MotivoPreparar, d.MotivoCancelar, d.MotivoListar, d.MotivoDetalle} {
		if motivo.Validar() != nil {
			return nil, ErrContactoPropioNoDisponible
		}
	}
	repo, err := postgres.NuevoRegistroOperacionContactoPostgreSQL(s.dependencias.PoolEscritor)
	if err != nil {
		return nil, ErrContactoPropioNoDisponible
	}
	copia := *s
	copia.registroOperaciones = repo
	return &ServicioOperaciones{Servicio: &copia, repositorio: repo, dependenciasOperaciones: d}, nil
}

func (s *ServicioOperaciones) prepararAplicacion(ctx context.Context) (*application.ServicioOperacionesContactoUsuario, domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2, domain.RecursoAutorizable, domain.ReferenciaCorrelacionAutorizacionV2, error) {
	var vinculo domain.VinculoAutenticacionActorV2
	var resultado domain.ResultadoContextoActorRegistradoV2
	var recurso domain.RecursoAutorizable
	var correlacion domain.ReferenciaCorrelacionAutorizacionV2
	if s == nil || s.Servicio == nil || ctx == nil || ctx.Err() != nil || dependenciaContactoPropioNula(s.repositorio) {
		return nil, vinculo, resultado, recurso, correlacion, ErrContactoPropioNoDisponible
	}
	vinculo, resultado, err := s.resolverContactoPropio(ctx)
	if err != nil {
		return nil, vinculo, resultado, recurso, correlacion, application.ErrOperacionContactoAccesoDenegado
	}
	d := s.dependencias
	correlacion, err = domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, d.GeneradorCorrelacion)
	if err != nil {
		return nil, vinculo, resultado, recurso, correlacion, ErrContactoPropioNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return nil, vinculo, resultado, recurso, correlacion, ErrContactoPropioNoDisponible
	}
	recurso = domain.RecursoAutorizable{Referencia: resultado.Contexto.PersonaRef, ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: clonarAmbitos(d.AmbitosRecurso)}
	preparador := preparadorAuditoria{fuente: d.FuenteAutorizacion, seudonimizador: d.Seudonimizador, reloj: d.Reloj, correlacion: correlacionRef, recurso: recurso}
	servicio, err := application.NuevoServicioOperacionesContactoUsuario(preparador, d.Huellas, emisorOperacionesContacto{s.dependenciasOperaciones}, s.repositorio, generadorOperacionContacto{})
	if err != nil {
		return nil, vinculo, resultado, recurso, correlacion, ErrContactoPropioNoDisponible
	}
	return servicio, vinculo, resultado, recurso, correlacion, nil
}

func (s *ServicioOperaciones) PrepararOperacion(ctx context.Context, correo string, versionEsperada uint64) (ports.OperacionContactoUsuario, error) {
	return s.prepararOperacion(ctx, correo, versionEsperada, "", "")
}

func (s *ServicioOperaciones) prepararOperacion(ctx context.Context, correo string, versionEsperada uint64, operacionRef, sujetoEsperado string) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if correo == "" || len(correo) > 254 || strings.TrimSpace(correo) != correo || strings.ContainsAny(correo, "\r\n") || versionEsperada >= 1<<53-1 {
		return vacio, ErrContactoPropioInvalido
	}
	if operacionRef != "" && (!application.ReferenciaOperacionContactoValida(operacionRef) || !domain.ReferenciaSujetoContactoUsuarioValida(sujetoEsperado)) {
		return vacio, ErrContactoPropioNoDisponible
	}
	servicio, vinculo, resultado, recurso, correlacion, err := s.prepararAplicacion(ctx)
	if err != nil {
		return vacio, err
	}
	if sujetoEsperado != "" && resultado.Contexto.PersonaRef != sujetoEsperado {
		return vacio, application.ErrOperacionContactoAccesoDenegado
	}
	op, err := servicio.Preparar(ctx, ports.SolicitudPrepararOperacionContacto{ContextoActor: resultado.Contexto, Correo: correo, OperacionRef: operacionRef, VersionEsperada: versionEsperada,
		Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: s.dependenciasOperaciones.MotivoPreparar, Accion: application.AccionPrepararOperacionContacto,
			Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if err != nil {
		return op, traducirErrorOperacion(err)
	}
	return op, nil
}

// El registro propio entrega un op_ref durable y una persona acreditada. La
// preparación usa ese op_ref exacto; solo entonces se confirma el contacto y
// se pide al propietario del alta su transición durable a alta_completa.
func (s *ServicioOperaciones) CompletarContactoDeAlta(ctx context.Context, alta ports.ReferenciaAltaContactoUsuario, correo string, confirmar ports.ConfirmadorAltaContactoUsuario) (ports.ConfirmacionAltaContactoUsuario, error) {
	vacio := ports.ConfirmacionAltaContactoUsuario{}
	if s == nil || !application.ReferenciaOperacionContactoValida(alta.OperacionRef) || !domain.ReferenciaSujetoContactoUsuarioValida(alta.PersonaRef) || dependenciaContactoPropioNula(confirmar) {
		return vacio, ErrContactoPropioNoDisponible
	}
	preparada, err := s.prepararOperacion(ctx, correo, 0, alta.OperacionRef, alta.PersonaRef)
	if err != nil {
		return vacio, err
	}
	if preparada.OperacionRef != alta.OperacionRef || preparada.VersionEsperada != 0 ||
		(preparada.Estado != ports.OperacionContactoPreparada && preparada.Estado != ports.OperacionContactoConfirmada) {
		return vacio, ErrContactoPropioNoDisponible
	}
	return s.Servicio.CompletarContactoDeAlta(ctx, alta, correo, confirmar)
}

func (s *ServicioOperaciones) CancelarOperacion(ctx context.Context, operacionRef string) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if !application.ReferenciaOperacionContactoValida(operacionRef) {
		return vacio, ErrContactoPropioInvalido
	}
	servicio, vinculo, resultado, recurso, correlacion, err := s.prepararAplicacion(ctx)
	if err != nil {
		return vacio, err
	}
	op, err := servicio.Cancelar(ctx, ports.SolicitudGestionOperacionContacto{ContextoActor: resultado.Contexto, OperacionRef: operacionRef,
		Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: s.dependenciasOperaciones.MotivoCancelar, Accion: application.AccionCancelarOperacionContacto,
			Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if err != nil {
		return vacio, traducirErrorOperacion(err)
	}
	return op, nil
}

func (s *ServicioOperaciones) ListarOperaciones(ctx context.Context, limite uint32, despuesDe string) (ports.ResultadoListaOperacionesContacto, error) {
	vacio := ports.ResultadoListaOperacionesContacto{}
	if limite < 1 || limite > 50 || despuesDe != "" && !application.ReferenciaOperacionContactoValida(despuesDe) {
		return vacio, ErrContactoPropioInvalido
	}
	servicio, vinculo, resultado, recurso, correlacion, err := s.prepararAplicacion(ctx)
	if err != nil {
		return vacio, err
	}
	r, err := servicio.Listar(ctx, ports.SolicitudGestionOperacionContacto{ContextoActor: resultado.Contexto, Limite: limite, DespuesDe: despuesDe,
		Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: s.dependenciasOperaciones.MotivoListar, Accion: application.AccionListarOperacionesContacto,
			Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if err != nil {
		return vacio, traducirErrorOperacion(err)
	}
	return r, nil
}

func (s *ServicioOperaciones) DetalleOperacion(ctx context.Context, operacionRef string) (ports.ResultadoDetalleOperacionContacto, error) {
	vacio := ports.ResultadoDetalleOperacionContacto{}
	if !application.ReferenciaOperacionContactoValida(operacionRef) {
		return vacio, ErrContactoPropioInvalido
	}
	servicio, vinculo, resultado, recurso, correlacion, err := s.prepararAplicacion(ctx)
	if err != nil {
		return vacio, err
	}
	r, err := servicio.Detalle(ctx, ports.SolicitudGestionOperacionContacto{ContextoActor: resultado.Contexto, OperacionRef: operacionRef,
		Recurso: recurso, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: s.dependenciasOperaciones.MotivoDetalle, Accion: application.AccionDetalleOperacionContacto,
			Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if err != nil {
		return vacio, traducirErrorOperacion(err)
	}
	return r, nil
}

func traducirErrorOperacion(err error) error {
	switch {
	case errors.Is(err, application.ErrOperacionContactoPreparada), errors.Is(err, application.ErrOperacionContactoNoEncontrada), errors.Is(err, application.ErrOperacionContactoAccesoDenegado):
		return err
	case errors.Is(err, application.ErrContactoUsuarioConflicto):
		return ErrContactoPropioConflicto
	case errors.Is(err, application.ErrContactoUsuarioCommitIncierto):
		return ErrContactoPropioCommitIncierto
	default:
		return ErrContactoPropioNoDisponible
	}
}

// ConfirmarOperacion consume sólo una intención preparada por Contacto3. La
// referencia se compromete en el recurso V3; el adaptador PostgreSQL compara
// sujeto, versión y HMAC antes de insertar la versión y el recibo original.
func (s *Servicio) ConfirmarOperacion(ctx context.Context, operacionRef, correo string, versionEsperada uint64) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if s == nil || !application.ReferenciaOperacionContactoValida(operacionRef) || versionEsperada >= 1<<53-1 || dependenciaContactoPropioNula(s.registroOperaciones) {
		return vacio, ErrContactoPropioNoDisponible
	}
	recibo, err := s.guardar(ctx, correo, versionEsperada, "", operacionRef, s.registroOperaciones)
	if err != nil {
		return vacio, err
	}
	op := ports.OperacionContactoUsuario{OperacionRef: operacionRef, Estado: ports.OperacionContactoConfirmada, VersionEsperada: versionEsperada, Version: recibo.Version, ReciboRef: recibo.EvidenciaCentral.Referencia, ReplayConfirmado: recibo.ReplayConfirmado}
	if application.ValidarOperacionContacto(op) != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return op, nil
}
