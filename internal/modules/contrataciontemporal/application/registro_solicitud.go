// Package application orquesta los casos de uso del expediente de
// contratación temporal sin conocer HTTP, PostgreSQL ni el proveedor de
// identidad.
package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioRegistroInvalido  = errors.New("contratacion temporal: servicio de registro invalido")
	ErrSolicitudRegistroInvalida = errors.New(
		"contratacion temporal: solicitud de registro invalida",
	)
	ErrResultadoRegistroNoConfiable = errors.New(
		"contratacion temporal: resultado de registro no confiable",
	)
)

type SolicitudRegistrarExpediente struct {
	SesionRef             string
	PerfilRef             string
	CorrelacionRef        string
	MotivoAutorizacionRef string
	OrganizacionRef       string
	ClaveIdempotencia     string
	Solicitud             domain.SolicitudCentro
}

type ServicioRegistroSolicitud struct {
	identidades   ports.ResolutorIdentidadOperacion
	flujos        ports.ResolutorFlujoAlta
	huellas       ports.DerivadorHuellaAlta
	preparaciones ports.PreparadorAltaIdempotente
	autorizador   ports.AutorizadorAltaExpediente
	reloj         ports.Reloj
	transaccion   ports.TransaccionAltas
}

func NuevoServicioRegistroSolicitud(
	identidades ports.ResolutorIdentidadOperacion,
	flujos ports.ResolutorFlujoAlta,
	huellas ports.DerivadorHuellaAlta,
	preparaciones ports.PreparadorAltaIdempotente,
	autorizador ports.AutorizadorAltaExpediente,
	reloj ports.Reloj,
	transaccion ports.TransaccionAltas,
) (*ServicioRegistroSolicitud, error) {
	if dependenciaNula(identidades) || dependenciaNula(flujos) ||
		dependenciaNula(huellas) || dependenciaNula(preparaciones) ||
		dependenciaNula(autorizador) || dependenciaNula(reloj) ||
		dependenciaNula(transaccion) {
		return nil, ErrServicioRegistroInvalido
	}
	return &ServicioRegistroSolicitud{
		identidades: identidades, flujos: flujos, huellas: huellas,
		preparaciones: preparaciones, autorizador: autorizador,
		reloj: reloj, transaccion: transaccion,
	}, nil
}

func (s *ServicioRegistroSolicitud) Registrar(
	ctx context.Context,
	solicitud SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	if ctx == nil || s == nil || dependenciaNula(s.identidades) ||
		dependenciaNula(s.flujos) || dependenciaNula(s.huellas) ||
		dependenciaNula(s.preparaciones) || dependenciaNula(s.autorizador) ||
		dependenciaNula(s.reloj) || dependenciaNula(s.transaccion) {
		return ports.ReciboAlta{}, ErrServicioRegistroInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	if validarSolicitudRegistro(solicitud, instante) != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ErrSolicitudRegistroInvalida,
		)
	}
	solicitudCentro, err := solicitud.Solicitud.Clonar()
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ErrSolicitudRegistroInvalida,
			err,
		)
	}

	resolverIdentidad := ports.SolicitudResolverIdentidad{
		SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef,
		CorrelacionRef: solicitud.CorrelacionRef,
	}
	identidad, err := s.identidades.ResolverIdentidadOperacion(ctx, resolverIdentidad)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	datosIdentidad, err := identidad.Datos()
	if err != nil || datosIdentidad.PerfilRef != solicitud.PerfilRef ||
		!identidad.VigenteEn(instante) {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrIdentidadOperacionInvalida,
			err,
		)
	}

	resolverFlujo := ports.SolicitudResolverFlujo{
		OrganizacionRef: solicitud.OrganizacionRef,
		CentroRef:       solicitudCentro.CentroRef,
		CategoriaRef:    solicitudCentro.CategoriaRef,
		MotivoClave:     solicitudCentro.MotivoClave,
		Instante:        instante,
	}
	if resolverFlujo.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrFlujoNoDisponible
	}
	configuracion, err := s.flujos.ResolverFlujoAlta(ctx, resolverFlujo)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if configuracion.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrFlujoNoDisponible
	}

	solicitudParaHuella, err := solicitudCentro.Clonar()
	if err != nil {
		return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
	}
	materialHuella := ports.MaterialHuellaAlta{
		OrganizacionRef: solicitud.OrganizacionRef,
		ActorRef:        datosIdentidad.ActorRef, PerfilRef: datosIdentidad.PerfilRef,
		Flujo: configuracion.Flujo, Solicitud: solicitudParaHuella,
	}
	if materialHuella.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	huellas, err := s.huellas.DerivarHuellaAlta(ctx, materialHuella)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	preparar := ports.SolicitudPrepararAlta{
		ClaveIdempotencia:   solicitud.ClaveIdempotencia,
		HuellasPeticionHMAC: huellas,
		OrganizacionRef:     solicitud.OrganizacionRef,
		ActorRef:            datosIdentidad.ActorRef,
		PerfilRef:           datosIdentidad.PerfilRef,
	}
	if preparar.Validar() != nil {
		return ports.ReciboAlta{}, errors.Join(
			ErrSolicitudRegistroInvalida,
			ports.ErrPreparacionAltaInvalida,
		)
	}
	preparacion, err := s.preparaciones.PrepararAlta(ctx, preparar)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if preparacion.ValidarPara(preparar) != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	if preparacion.Estado == ports.PreparacionConfirmada {
		return *preparacion.ReciboConfirmado, nil
	}

	recurso := ports.RecursoAltaExpediente{
		ExpedienteRef:   preparacion.Referencias.ExpedienteRef,
		OrganizacionRef: solicitud.OrganizacionRef,
		CentroRef:       solicitudCentro.CentroRef,
		CategoriaRef:    solicitudCentro.CategoriaRef,
		FlujoRef:        configuracion.Flujo.DefinicionRef,
		FlujoVersion:    configuracion.Flujo.Version,
	}
	instanteAutorizacion := instanteCanonico(s.reloj.Ahora())
	autorizar := ports.SolicitudAutorizarAlta{
		Identidad: identidad, Recurso: recurso,
		MotivoRef:      solicitud.MotivoAutorizacionRef,
		CorrelacionRef: solicitud.CorrelacionRef,
		SolicitadaEn:   instanteAutorizacion,
	}
	if autorizar.Validar() != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrAutorizacionEfectoInvalida,
		)
	}
	autorizacion, err := s.autorizador.AutorizarAltaExpediente(ctx, autorizar)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instanteAlta := instanteCanonico(s.reloj.Ahora())
	datosAutorizacion, err := autorizacion.Datos()
	if err != nil || datosAutorizacion.RecursoRef != recurso.ExpedienteRef ||
		datosAutorizacion.ActorRef != datosIdentidad.ActorRef ||
		datosAutorizacion.PerfilRef != datosIdentidad.PerfilRef ||
		!identidad.VigenteEn(instanteAlta) ||
		datosAutorizacion.EmitidaEn.After(instanteAlta) ||
		!instanteAlta.Before(datosAutorizacion.ValidaHasta) {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrAutorizacionEfectoInvalida,
			err,
		)
	}

	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      preparacion.Referencias.ExpedienteRef,
		OrganizacionRef: solicitud.OrganizacionRef,
		NumeroVisible:   preparacion.Referencias.NumeroVisible,
		Flujo:           configuracion.Flujo,
		FaseInicial:     configuracion.FaseInicial,
		Solicitud:       solicitudCentro,
		Actuacion: domain.DatosActuacion{
			AccionClave:   configuracion.AccionInicial,
			ActorRef:      datosIdentidad.ActorRef,
			UnidadRef:     configuracion.UnidadInicialRef,
			ReciboRef:     preparacion.Referencias.ReciboRef,
			RealizadaEn:   instanteAlta,
			FaseDestino:   configuracion.FaseInicial,
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ErrSolicitudRegistroInvalida, err)
	}
	orden, err := ports.NuevaOrdenConfirmarAlta(ports.DatosOrdenConfirmarAlta{
		Expediente: expediente, Identidad: identidad,
		Autorizacion: autorizacion, Preparacion: preparacion,
		CorrelacionRef: solicitud.CorrelacionRef,
	})
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	recibo, err := s.transaccion.ConfirmarAlta(ctx, orden)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	// La transacción es la frontera de efecto. Una cancelación observada tras
	// un COMMIT confirmado no puede convertir el éxito durable en un fallo
	// ambiguo y provocar que el cliente repita la operación.
	if recibo.ValidarPara(expediente) != nil {
		return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
	}
	return recibo, nil
}

func validarSolicitudRegistro(
	solicitud SolicitudRegistrarExpediente,
	instante time.Time,
) error {
	resolverIdentidad := ports.SolicitudResolverIdentidad{
		SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef,
		CorrelacionRef: solicitud.CorrelacionRef,
	}
	if !domain.InstanteUTCCanonico(instante) ||
		resolverIdentidad.Validar() != nil ||
		!domain.ReferenciaOpacaValida(solicitud.MotivoAutorizacionRef) ||
		!domain.ReferenciaOpacaValida(solicitud.OrganizacionRef) ||
		solicitud.Solicitud.Validar() != nil {
		return ErrSolicitudRegistroInvalida
	}
	return nil
}

func instanteCanonico(valor time.Time) time.Time {
	if valor.IsZero() {
		return time.Time{}
	}
	return valor.UTC().Truncate(time.Microsecond)
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
