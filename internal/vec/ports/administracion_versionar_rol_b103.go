package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ResultadoPropuestaVersionarRolBolsaB103 distingue recuperación auditada de
// primera propuesta vigente. Replay sólo puede declararlo la autoridad SQL.
type ResultadoPropuestaVersionarRolBolsaB103 struct {
	Propuesta          domain.PropuestaVersionarRolBolsaB103
	Replay             bool
	AuditoriaAccesoRef string
}

func (r ResultadoPropuestaVersionarRolBolsaB103) ValidarPara(
	o domain.OrdenPropuestaVersionarRolBolsaB103, ahora time.Time) error {
	if r.Propuesta.ValidarPara(o) != nil || !referenciaAuditoriaGobiernoRol(r.AuditoriaAccesoRef) ||
		(!r.Replay && !r.Propuesta.CaducaEn.After(ahora)) {
		return domain.ErrVersionarRolBolsaB103Invalido
	}
	return nil
}

// AutoridadVersionarRolBolsaB103 es nominal y durable. Resolver devuelve el
// catálogo de la fuente central; las preimágenes de asignación son expectativas
// sin autoridad que SQL debe cotejar bajo bloqueo. Proponer no publica. Cerrar
// recupera la propuesta original por referencia y consume V3 en la misma
// transacción que publicación, CAS de las asignaciones seleccionadas, historia,
// auditoría y recibo. Un fallo de cualquier parte revierte el efecto completo.
type AutoridadVersionarRolBolsaB103 interface {
	ResolverCatalogoVersionarRolBolsaB103(context.Context,
		domain.SolicitudPropuestaVersionarRolBolsaB103) (domain.CatalogoAccionesAdministracionV1, error)
	ProponerVersionarRolBolsaB103(context.Context,
		domain.OrdenPropuestaVersionarRolBolsaB103) (ResultadoPropuestaVersionarRolBolsaB103, error)
	CerrarVersionarRolBolsaB103(context.Context,
		domain.SolicitudCierreVersionarRolBolsaB103) (domain.CierreVersionarRolBolsaB103, error)
}
