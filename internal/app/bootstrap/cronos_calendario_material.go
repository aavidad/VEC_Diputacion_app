package bootstrap

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// El calendario usa el gobierno común. El lote no publica concesiones de
// negocio y conserva las audiencias nominales de Personal y Organización.
func publicarMaterialCalendarioCronosEnLote(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
) ([2]*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	var vacio [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	if ctx == nil || ctx.Err() != nil || gobierno == nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	descriptores := [2]descriptorMaterialConsumidorV3Desarrollo{descriptorMaterialUnidadTrabajoC4(), descriptorMaterialOrganizacionHistorica()}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores[:])
	if err != nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	var derivados [2]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range derivados {
			borrarBytes(derivados[i].claveHMAC)
		}
	}()
	for i, descriptor := range descriptores {
		if _, ok := catalogo.descriptorPara(descriptor.Audiencia); !ok {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
		derivados[i], err = derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	var proveedores [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	err = ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range derivados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &derivados[i]); err != nil {
				return err
			}
			var err error
			proveedores[i], err = nuevoProveedorMaterialAutorizacionBaseDesarrollo(derivados[i], reloj)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	return proveedores, nil
}
