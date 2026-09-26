package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func mismaRecepcionCargaDocumental(
	existente, preparada domain.CargaDocumental, contenido domain.ContenidoCargaDocumental,
) bool {
	if existente.Validar() != nil || preparada.Validar() != nil ||
		existente.Estado != domain.EstadoCargaDocumentalCuarentena ||
		existente.Version != preparada.Version+1 ||
		existente.ID != preparada.ID ||
		existente.IndiceIdempotenciaHMAC != preparada.IndiceIdempotenciaHMAC ||
		existente.HuellaSolicitudHMAC != preparada.HuellaSolicitudHMAC ||
		existente.ContenidoCuarentena == nil {
		return false
	}
	guardado := existente.ContenidoCuarentena
	return guardado.ConectorID == contenido.ConectorID &&
		guardado.Referencia == contenido.Referencia &&
		guardado.Version == contenido.Version &&
		guardado.Zona == contenido.Zona &&
		guardado.MIME == contenido.MIME &&
		guardado.Tamano == contenido.Tamano &&
		guardado.HuellaSHA256 == contenido.HuellaSHA256
}

func contextoPersistenciaCarga(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
}

func errorConfirmacionCargaDocumentalPendiente(causa error) error {
	if causa == nil {
		causa = ports.ErrConfirmacionCargaDirectaNoDisponible
	}
	return errors.Join(ports.ErrConfirmacionCargaDocumentalPendiente, causa)
}
