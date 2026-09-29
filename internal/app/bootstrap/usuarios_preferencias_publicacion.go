package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type proveedoresMaterialPreferenciasUsuarios [4]*proveedorMaterialAltaContratacionTemporalDesarrollo

type transaccionGobiernoPreferencias func(context.Context, *pgxpool.Pool, func(pgx.Tx) error) error
type publicarGobiernoPreferencias func(context.Context, pgx.Tx, *materialAtestacionContratacionTemporalDesarrollo) error

func ejecutarPublicacionPreferenciasEnUnaTx(ctx context.Context, pool *pgxpool.Pool, materiales *[4]materialAtestacionContratacionTemporalDesarrollo,
	ejecutar transaccionGobiernoPreferencias, publicar publicarGobiernoPreferencias) error {
	if ctx == nil || materiales == nil || ejecutar == nil || publicar == nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	return ejecutar(ctx, pool, func(tx pgx.Tx) error {
		for i := range materiales {
			if err := publicar(ctx, tx, &materiales[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// Las cuatro audiencias forman un único corte: si una publicación falla, la
// transacción de gobierno revierte todas. Ningún proveedor se entrega hasta
// que el COMMIT de las cuatro claves haya terminado.
func publicarMaterialPreferenciasUsuariosEnLote(ctx context.Context, gobierno *pgxpool.Pool, base materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo,
) (proveedoresMaterialPreferenciasUsuarios, error) {
	vacios := proveedoresMaterialPreferenciasUsuarios{}
	if ctx == nil || gobierno == nil {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	descriptores := descriptoresMaterialPreferenciasUsuariosDesarrollo()
	if len(descriptores) != len(vacios) {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	var preparados [4]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range preparados {
			borrarBytes(preparados[i].claveHMAC)
		}
	}()
	for i, d := range descriptores {
		declarado, ok := catalogo.descriptorPara(d.Audiencia)
		if !ok || declarado != d {
			return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		derivado, err := derivarMaterialConsumidorV3Desarrollo(base, d)
		if err != nil {
			return vacios, err
		}
		preparados[i] = derivado
	}
	err := ejecutarPublicacionPreferenciasEnUnaTx(ctx, gobierno, &preparados, ejecutarTransaccionGobiernoCTDesarrollo, publicarGobiernoAtestacionCTEnTxDesarrollo)
	if err != nil {
		return vacios, err
	}
	var resultado proveedoresMaterialPreferenciasUsuarios
	for i := range preparados {
		p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(preparados[i], reloj)
		if err != nil {
			return vacios, err
		}
		resultado[i] = p
	}
	return resultado, nil
}
