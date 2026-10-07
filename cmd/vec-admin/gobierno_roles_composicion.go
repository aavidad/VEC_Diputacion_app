package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

func fuenteCatalogoGobiernoOficial(ctx context.Context,
	pool *pgxpool.Pool) (ports.FuenteCatalogoAccionesAdministracionV1, error) {
	return pg.NuevaFuenteCatalogoAcciones(ctx, pool)
}

// componerGobiernoRolesADMIN consume exclusivamente el catálogo AUT58, el
// LOGIN Gov y las dos capacidades V3 privadas. La fuente la construye el
// caller desde el pool lector exacto; ningún archivo HTTP aporta permisos.
func componerGobiernoRolesADMIN(ctx context.Context, base configuracionPerfilesPrivada,
	c configuracionGobiernoRolesPrivada, poolFuente, poolRegistro, poolMotivos,
	poolGobierno, poolCatalogo *pgxpool.Pool,
	fuente ports.FuenteCatalogoAccionesAdministracionV1,
	firmante ports.FirmanteAtestacionesAutorizacionV3,
	reloj ports.Reloj) (*administracion.MontajeGobiernoRolNuevoADMIN, error) {
	if ctx == nil || ctx.Err() != nil || poolGobierno == nil || poolCatalogo == nil ||
		poolGobierno == poolCatalogo || fuente == nil ||
		acreditarPoolCentral(ctx, poolGobierno, "vec_admin_gobierno_roles_ejecutor") != nil ||
		acreditarPoolCentral(ctx, poolCatalogo, "vec_admin_catalogo_acciones_lector") != nil {
		return nil, errorArranque("gobierno_roles_pool")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("gobierno_roles_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("gobierno_roles_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	servicio, err := administracion.NuevoServicioGobiernoRolNuevoV3(ctx,
		administracion.DependenciasGobiernoRolNuevoV3{PoolGobierno: poolGobierno,
			FuenteCatalogo: fuente, Confianza: conf,
			DependenciasPDP: administracion.DependenciasConfianzaPerfilesV3{
				PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos,
				CatalogoMotivosID: base.CatalogoMotivosID, Firmante: firmante, Reloj: reloj,
				Generador:        seguridad.GeneradorReferenciasCriptograficas{},
				VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second},
			Motivos: c.Motivos, Reloj: reloj})
	if err != nil {
		return nil, errorArranque("gobierno_roles_servicio")
	}
	return &administracion.MontajeGobiernoRolNuevoADMIN{Servicio: servicio, Fuente: fuente}, nil
}
