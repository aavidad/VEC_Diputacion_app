package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

func componerGobiernoInscripcionADMIN(ctx context.Context, base configuracionPerfilesPrivada,
	c configuracionGobiernoInscripcionPrivada, poolFuente, poolRegistro, poolMotivos,
	poolGobierno, poolCatalogo *pgxpool.Pool,
	fuente ports.FuenteCatalogoAccionesAdministracionV1,
	firmante ports.FirmanteAtestacionesAutorizacionV3,
	reloj ports.Reloj) (*administracion.MontajeGobiernoInscripcionADMIN, error) {
	if ctx == nil || ctx.Err() != nil || poolGobierno == nil || poolCatalogo == nil ||
		poolGobierno == poolCatalogo || fuente == nil ||
		acreditarPoolCentral(ctx, poolGobierno, "vec_admin_version_inscripcion_ejecutor") != nil ||
		acreditarPoolCentral(ctx, poolCatalogo, "vec_admin_catalogo_acciones_lector") != nil {
		return nil, errorArranque("inscripcion_gobierno_pool")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("inscripcion_gobierno_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("inscripcion_gobierno_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	servicio, err := administracion.NuevoServicioGobiernoInscripcionV3(ctx,
		administracion.DependenciasGobiernoInscripcionV3{PoolGobierno: poolGobierno,
			FuenteCatalogo: fuente, Confianza: conf,
			DependenciasPDP: administracion.DependenciasConfianzaPerfilesV3{
				PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos,
				CatalogoMotivosID: base.CatalogoMotivosID, Firmante: firmante, Reloj: reloj,
				Generador:        seguridad.GeneradorReferenciasCriptograficas{},
				VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second},
			Motivos: c.Motivos, Reloj: reloj})
	if err != nil {
		return nil, errorArranque("inscripcion_gobierno_servicio")
	}
	return &administracion.MontajeGobiernoInscripcionADMIN{Servicio: servicio, Fuente: fuente}, nil
}
