package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrOrdenIntentoAuditoriaInvalida = domain.ErrOrdenIntentoAuditoriaInvalida
	ErrAcuseIntentoAuditoriaInvalido = domain.ErrAcuseIntentoAuditoriaInvalido
	ErrIntentoAuditoriaNoDisponible  = domain.ErrIntentoAuditoriaNoDisponible
	ErrIntentoAuditoriaConflicto     = domain.ErrIntentoAuditoriaConflicto
)

type OrdenIntentoAuditoria = domain.OrdenIntentoAuditoria
type DatosOrdenIntentoAuditoria = domain.DatosOrdenIntentoAuditoria
type AcuseIntentoAuditoria = domain.AcuseIntentoAuditoria

// NuevaReferenciaIntentoAuditoria emite una clave opaca del servidor.
func NuevaReferenciaIntentoAuditoria() (string, error) {
	return domain.NuevaReferenciaIntentoAuditoria()
}

func NuevaOrdenIntentoAuditoria(
	intentoRef string,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	vinculo domain.VinculoAutenticacionActorV2,
	datos domain.DatosIntentoAuditoria,
) (OrdenIntentoAuditoria, error) {
	return domain.NuevaOrdenIntentoAuditoria(intentoRef, resultadoContexto, vinculo, datos)
}

// RegistradorIntentosAuditoria persiste un resultado denegado o error una vez
// cerrado el intento original. El adaptador usa una transacción propia y
// devuelve acuse únicamente después de confirmar COMMIT.
type RegistradorIntentosAuditoria interface {
	AppendIntentoAuditoria(context.Context, OrdenIntentoAuditoria) (AcuseIntentoAuditoria, error)
}
