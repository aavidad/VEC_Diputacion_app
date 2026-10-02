package composicion

import (
	"bytes"
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// RegistroIntentosLectorRelacionRPT conserva únicamente un actor cotejado con
// la identidad actual registrada. Un actor declarado o caducado no se registra
// como acreditado. La ausencia de identidad deja ese dato vacío, sin inventarlo.
type RegistroIntentosLectorRelacionRPT struct {
	identidad ResolutorIdentidadLectorRelacionRPT
	destino   ports.DestinoIntentosLectorRelacionRPT
	ahora     func() time.Time
}

func NuevoRegistroIntentosLectorRelacionRPT(i ResolutorIdentidadLectorRelacionRPT, d ports.DestinoIntentosLectorRelacionRPT, ahora func() time.Time) (*RegistroIntentosLectorRelacionRPT, error) {
	if dependenciaNula(i) || dependenciaNula(d) || ahora == nil {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	return &RegistroIntentosLectorRelacionRPT{i, d, ahora}, nil
}
func (r *RegistroIntentosLectorRelacionRPT) VerificarRegistroRelacionRPT(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.destino) || r.ahora == nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	if err := r.destino.VerificarDestinoRelacionRPT(ctx); err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}
func (r *RegistroIntentosLectorRelacionRPT) RegistrarIntentoRelacionRPT(ctx context.Context, in ports.IntentoLectorRelacionRPT) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.identidad) || dependenciaNula(r.destino) || r.ahora == nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	evento := ports.EventoIntentoLectorRelacionRPT{CorrelacionRef: valor, Motivo: in.Motivo}
	if domain.ReferenciaRelacionValida(in.RelacionRef) {
		evento.RelacionRef = in.RelacionRef
	}
	identidad, err := r.identidad.ResolverIdentidadLectorRelacionRPT(ctx)
	instanteActual := r.ahora().UTC().Truncate(time.Microsecond)
	if err == nil && identidad.Resultado.Validar() == nil && identidad.Vinculo.ValidarPara(identidad.Resultado) == nil && identidad.Vinculo.VigenteEn(instanteActual, identidad.Resultado) {
		canon, canonErr := in.Actor.RepresentacionCanonicaVinculadaV2()
		if canonErr == nil && bytes.Equal(canon, identidad.Resultado.RepresentacionCanonica) {
			evento.ActorRef = identidad.Resultado.Contexto.PersonaRef
		}
	}
	if err := r.destino.RegistrarEventoRelacionRPT(ctx, evento); err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}

var _ ports.RegistroIntentosLectorRelacionRPT = (*RegistroIntentosLectorRelacionRPT)(nil)
