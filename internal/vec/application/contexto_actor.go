package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioContextoActor resuelve la cuenta tecnica autenticada a una unica
// persona canonica para el perfil solicitado expresamente. No autentica, no
// autoriza y no infiere perfiles; produce el contexto cerrado que consumiran
// despues los casos de uso y el PDP.
type ServicioContextoActor struct {
	modo      modoServicioContextoActor
	resolutor ports.ResolutorRegistroContextoActorV1
	generador ports.GeneradorOperacionContextoActorV1
	fuente    ports.FuenteContextoActor
	reloj     ports.Reloj
}

type modoServicioContextoActor uint8

const (
	modoServicioContextoActorProductivoV1 modoServicioContextoActor = iota + 1
	modoServicioContextoActorFuenteHeredada
)

// NuevoServicioContextoActor conserva la API heredada para pruebas y
// migracion. No debe componerse en produccion: FuenteContextoActor no puede
// demostrar que la capacidad fue registrada en la misma transaccion.
func NuevoServicioContextoActor(
	fuente ports.FuenteContextoActor,
	reloj ports.Reloj,
) (*ServicioContextoActor, error) {
	if dependenciaContextoActorNula(fuente) || dependenciaContextoActorNula(reloj) {
		return nil, domain.ErrContextoActorInvalido
	}
	return &ServicioContextoActor{
		modo: modoServicioContextoActorFuenteHeredada, fuente: fuente, reloj: reloj,
	}, nil
}

// NuevoServicioContextoActorProductivoV1 exige el puerto que resuelve y
// registra de forma atomica. No admite FuenteContextoActor ni cae a la ruta
// heredada.
func NuevoServicioContextoActorProductivoV1(
	resolutor ports.ResolutorRegistroContextoActorV1,
	generador ports.GeneradorOperacionContextoActorV1,
	reloj ports.Reloj,
) (*ServicioContextoActor, error) {
	if dependenciaContextoActorNula(resolutor) || dependenciaContextoActorNula(generador) ||
		dependenciaContextoActorNula(reloj) {
		return nil, domain.ErrContextoActorInvalido
	}
	return &ServicioContextoActor{
		modo: modoServicioContextoActorProductivoV1, resolutor: resolutor,
		generador: generador, reloj: reloj,
	}, nil
}

func (s *ServicioContextoActor) Resolver(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) (domain.ContextoActor, error) {
	if ctx == nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(domain.ErrSolicitudContextoActorInvalida)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	if s == nil || dependenciaContextoActorNula(s.reloj) {
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrResolutorRegistroContextoActorNoDisponible)
	}
	if err := solicitud.Validar(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}

	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() {
		return domain.ContextoActor{}, errorResolucionContextoActor(domain.ErrContextoActorInvalido)
	}
	switch s.modo {
	case modoServicioContextoActorProductivoV1:
		return s.resolverYRegistrarProductivoV1(ctx, solicitud, instante)
	case modoServicioContextoActorFuenteHeredada:
		return s.resolverDesdeFuenteHeredada(ctx, solicitud, instante)
	default:
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrResolutorRegistroContextoActorNoDisponible)
	}
}

func (s *ServicioContextoActor) resolverYRegistrarProductivoV1(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
	instante time.Time,
) (domain.ContextoActor, error) {
	if dependenciaContextoActorNula(s.resolutor) || dependenciaContextoActorNula(s.generador) ||
		s.fuente != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrResolutorRegistroContextoActorNoDisponible)
	}
	operacionRef, err := s.generador.NuevaReferenciaOperacionContextoActorV1(ctx)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return domain.ContextoActor{}, errorResolucionContextoActor(contextoErr)
		}
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrGeneradorOperacionContextoActorNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	solicitudRegistro := ports.SolicitudResolucionRegistroContextoActorV1{
		OperacionRef: operacionRef, Contexto: solicitud, SolicitadoEn: instante,
	}
	if solicitudRegistro.Validar() != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	confirmacion, err := s.resolutor.ResolverYRegistrarContextoActorV1(ctx, solicitudRegistro)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return domain.ContextoActor{}, errorResolucionContextoActor(contextoErr)
		}
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrResolutorRegistroContextoActorNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	// El resolutor puede haber esperado bloqueos. El instante previo a la
	// llamada no es autoridad: se contrasta el tiempo DB del recibo con ambos
	// extremos locales y se comprueba de nuevo la vigencia antes de entregar.
	comprobadoEn := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if confirmacion.ValidarPara(solicitudRegistro) != nil || comprobadoEn.IsZero() ||
		comprobadoEn.Before(instante) ||
		comprobadoEn.Sub(instante) > ports.VentanaMaximaFrescuraContextoActorV1 ||
		confirmacion.ResueltoEnAutoritativo.Before(instante) ||
		confirmacion.ResueltoEnAutoritativo.After(comprobadoEn) ||
		confirmacion.ResueltoEnAutoritativo.Sub(instante) > ports.VentanaMaximaFrescuraContextoActorV1 {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	copia, err := confirmacion.Contexto.Clonar()
	if err != nil || !contextoActorVigenteEn(copia, confirmacion.ResueltoEnAutoritativo) ||
		!contextoActorVigenteEn(copia, comprobadoEn) {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	return copia, nil
}

func contextoActorVigenteEn(actor domain.ContextoActor, instante time.Time) bool {
	if actor.Validar() != nil || instante.Before(actor.ResueltoEn) ||
		!actor.Instantanea.VigenteEn(instante) {
		return false
	}
	for _, vinculo := range actor.Instantanea.Vinculos {
		if !vinculo.VigenteEn(instante) {
			return false
		}
	}
	return true
}

func (s *ServicioContextoActor) resolverDesdeFuenteHeredada(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
	instante time.Time,
) (domain.ContextoActor, error) {
	if dependenciaContextoActorNula(s.fuente) || s.resolutor != nil || s.generador != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrFuenteContextoActorNoDisponible)
	}
	instantaneas, err := s.fuente.BuscarInstantaneasContextoActor(ctx, solicitud)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return domain.ContextoActor{}, errorResolucionContextoActor(contextoErr)
		}
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrFuenteContextoActorNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	// Cero y mas de una coincidencia reciben exactamente el mismo resultado. No
	// se revela existencia ni se elige la primera aunque apunte a la misma persona.
	if len(instantaneas) != 1 {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	instantanea := instantaneas[0]
	if instantanea.Validar() != nil ||
		instantanea.CuentaRef != solicitud.Cuenta.CuentaRef ||
		instantanea.PerfilActivoRef != solicitud.PerfilActivoRef ||
		!instantanea.VigenteEn(instante) {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}

	resultado, err := domain.NuevoContextoActor(solicitud.Cuenta, instantanea, instante)
	if err != nil || resultado.PerfilActivoRef != solicitud.PerfilActivoRef {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	copia, err := resultado.Clonar()
	if err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	return copia, nil
}

func errorResolucionContextoActor(causa error) error {
	if causa == nil {
		return domain.ErrContextoActorNoResuelto
	}
	return errors.Join(domain.ErrContextoActorNoResuelto, causa)
}

func dependenciaContextoActorNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
