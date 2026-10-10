package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ResultadoPropuestaVersionarRolBolsa distingue recuperación auditada de
// primera propuesta vigente. Replay sólo puede declararlo la autoridad SQL.
type ResultadoPropuestaVersionarRolBolsa struct {
	Propuesta          domain.PropuestaVersionarRolBolsa
	Replay             bool
	AuditoriaAccesoRef string
}

func (r ResultadoPropuestaVersionarRolBolsa) ValidarPara(
	o domain.OrdenPropuestaVersionarRolBolsa, ahora time.Time) error {
	if r.Propuesta.ValidarPara(o) != nil || !referenciaAuditoriaGobiernoRol(r.AuditoriaAccesoRef) ||
		(!r.Replay && !r.Propuesta.CaducaEn.After(ahora)) {
		return domain.ErrVersionarRolBolsaInvalido
	}
	return nil
}

// AutoridadVersionarRolBolsa es nominal y durable. Resolver devuelve el
// catálogo de la fuente central; las preimágenes de asignación son expectativas
// sin autoridad que SQL debe cotejar bajo bloqueo. Proponer no publica. Cerrar
// recupera la propuesta original por referencia y consume V3 en la misma
// transacción que publicación, CAS de las asignaciones seleccionadas, historia,
// auditoría y recibo. Un fallo de cualquier parte revierte el efecto completo.
type AutoridadVersionarRolBolsa interface {
	ResolverCatalogoVersionarRolBolsa(context.Context,
		domain.SolicitudPropuestaVersionarRolBolsa) (domain.CatalogoAccionesAdministracionV1, error)
	ProponerVersionarRolBolsa(context.Context,
		domain.OrdenPropuestaVersionarRolBolsa) (ResultadoPropuestaVersionarRolBolsa, error)
	CerrarVersionarRolBolsa(context.Context,
		domain.SolicitudCierreVersionarRolBolsa) (domain.CierreVersionarRolBolsa, error)
}
