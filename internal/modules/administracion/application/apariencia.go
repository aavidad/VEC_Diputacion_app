package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain"
	"vec-diputacion-granada/internal/modules/administracion/ports"
)

var (
	ErrAparienciaNoDisponible      = errors.New("administracion: apariencia no disponible")
	ErrAparienciaDenegada          = errors.New("administracion: acceso a apariencia denegado")
	ErrResultadoAparienciaIncierto = errors.New("administracion: resultado durable de apariencia indeterminado")
)

type ServicioApariencia struct {
	autorizador ports.AutorizadorApariencia
	registro    ports.RegistroApariencia
}

func NuevoServicioApariencia(a ports.AutorizadorApariencia, r ports.RegistroApariencia) (*ServicioApariencia, error) {
	if puertoNulo(a) || puertoNulo(r) {
		return nil, ErrAparienciaNoDisponible
	}
	return &ServicioApariencia{autorizador: a, registro: r}, nil
}

func (s *ServicioApariencia) ConsultarTemaGlobal(ctx context.Context) (ports.ResultadoConsultaApariencia, error) {
	vacio := ports.ResultadoConsultaApariencia{}
	if !s.disponible(ctx) {
		return vacio, ErrAparienciaNoDisponible
	}
	pedido := ports.SolicitudAutorizacionApariencia{
		Accion: ports.AccionConsultarApariencia, RecursoRef: ports.RecursoAparienciaGlobal,
		AmbitoRef: ports.AmbitoAparienciaGlobal, Finalidad: ports.FinalidadConsultarTema,
	}
	concesion, err := s.autorizador.ExigirApariencia(ctx, pedido)
	if err != nil || !concesionValida(concesion, pedido, time.Now().UTC()) {
		return vacio, ErrAparienciaDenegada
	}
	if ctx.Err() != nil {
		return vacio, ErrAparienciaNoDisponible
	}
	resultado, err := s.registro.ConsultarAuditable(ctx, ports.OrdenConsultaApariencia{Concesion: concesion})
	if err != nil {
		return vacio, ErrAparienciaNoDisponible
	}
	if resultado.Estado.Validar() != nil || resultado.DecisionRef != concesion.DecisionRef ||
		!domain.ReferenciaAparienciaValida(resultado.AuditoriaRef) ||
		!instanteVigente(resultado.ConsultadaEn, concesion) {
		return vacio, ErrResultadoAparienciaIncierto
	}
	return resultado, nil
}

func (s *ServicioApariencia) PublicarTemaGlobal(ctx context.Context, orden domain.OrdenPublicacion) (ports.ReciboPublicacion, error) {
	vacio := ports.ReciboPublicacion{}
	if !s.disponible(ctx) {
		return vacio, ErrAparienciaNoDisponible
	}
	if err := orden.Validar(); err != nil {
		return vacio, err
	}
	pedido := ports.SolicitudAutorizacionApariencia{
		Accion: ports.AccionPublicarApariencia, RecursoRef: ports.RecursoAparienciaGlobal,
		AmbitoRef: ports.AmbitoAparienciaGlobal, Finalidad: ports.FinalidadPublicarTema,
		Tema: orden.Tema, RevisionGlobalEsperada: orden.RevisionGlobalEsperada,
		ClaveIdempotencia: orden.ClaveIdempotencia,
	}
	concesion, err := s.autorizador.ExigirApariencia(ctx, pedido)
	if err != nil || !concesionValida(concesion, pedido, time.Now().UTC()) ||
		!domain.ReferenciaAparienciaValida(concesion.AprobacionRef) ||
		!domain.ReferenciaAparienciaValida(concesion.AprobadorRef) ||
		concesion.AprobadorRef == concesion.ActorRef {
		return vacio, ErrAparienciaDenegada
	}
	if ctx.Err() != nil {
		return vacio, ErrAparienciaNoDisponible
	}
	recibo, err := s.registro.PublicarAtomico(ctx, ports.OrdenConfirmarPublicacion{
		Publicacion: orden, Concesion: concesion,
	})
	if err != nil {
		if errors.Is(err, ports.ErrRevisionGlobalObsoleta) {
			return vacio, ports.ErrRevisionGlobalObsoleta
		}
		if errors.Is(err, ports.ErrClavePublicacionEnConflicto) {
			return vacio, ports.ErrClavePublicacionEnConflicto
		}
		return vacio, ErrResultadoAparienciaIncierto
	}
	siguiente, _ := orden.RevisionSiguiente()
	if recibo.Estado.Validar() != nil || !recibo.Estado.Configurado ||
		recibo.Estado.Tema != orden.Tema || recibo.Estado.RevisionGlobal != siguiente ||
		recibo.RevisionAnterior != orden.RevisionGlobalEsperada ||
		recibo.ClaveIdempotencia != orden.ClaveIdempotencia ||
		recibo.ActorRef != concesion.ActorRef || recibo.AprobacionRef != concesion.AprobacionRef ||
		!domain.ReferenciaAparienciaValida(recibo.DecisionRef) ||
		(!recibo.Replay && recibo.DecisionRef != concesion.DecisionRef) ||
		!domain.ReferenciaAparienciaValida(recibo.AuditoriaRef) ||
		!domain.ReferenciaAparienciaValida(recibo.EventoRef) ||
		recibo.ConfirmadaEn.IsZero() || recibo.ConfirmadaEn.Location() != time.UTC ||
		!recibo.ConfirmadaEn.Equal(recibo.Estado.PublicadaEn) ||
		(!recibo.Replay && !instanteVigente(recibo.ConfirmadaEn, concesion)) {
		return vacio, ErrResultadoAparienciaIncierto
	}
	return recibo, nil
}

func (s *ServicioApariencia) disponible(ctx context.Context) bool {
	return s != nil && ctx != nil && ctx.Err() == nil && !puertoNulo(s.autorizador) && !puertoNulo(s.registro)
}

func concesionValida(c ports.ConcesionApariencia, p ports.SolicitudAutorizacionApariencia, ahora time.Time) bool {
	return c.Permitida && c.Accion == p.Accion && c.RecursoRef == p.RecursoRef &&
		c.AmbitoRef == p.AmbitoRef && c.Finalidad == p.Finalidad &&
		c.Tema == p.Tema && c.RevisionGlobalEsperada == p.RevisionGlobalEsperada &&
		c.ClaveIdempotencia == p.ClaveIdempotencia &&
		domain.ReferenciaAparienciaValida(c.ActorRef) &&
		domain.ReferenciaAparienciaValida(c.PerfilRef) &&
		domain.ReferenciaAparienciaValida(c.DecisionRef) &&
		c.EmitidaEn.Location() == time.UTC && c.ExpiraEn.Location() == time.UTC &&
		!c.EmitidaEn.IsZero() && !c.ExpiraEn.IsZero() &&
		!ahora.Before(c.EmitidaEn) && ahora.Before(c.ExpiraEn)
}

func instanteVigente(instante time.Time, c ports.ConcesionApariencia) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		!instante.Before(c.EmitidaEn) && instante.Before(c.ExpiraEn)
}

func puertoNulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
