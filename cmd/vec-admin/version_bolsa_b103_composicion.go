package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

// B103 usa un ejecutor AUT75 v2 propio. La fuente AUT58 conserva su LOGIN
// lector y la cadena PDP/atestación ADMIN no publica capacidades al arrancar.
func componerVersionBolsaB103ADMIN(ctx context.Context, base configuracionPerfilesPrivada,
	c configuracionVersionBolsaB103Privada, poolFuente, poolRegistro, poolMotivos,
	poolGobierno, poolCatalogo *pgxpool.Pool,
	fuente ports.FuenteCatalogoAccionesAdministracionV1,
	firmante ports.FirmanteAtestacionesAutorizacionV3,
	reloj ports.Reloj) (*administracion.MontajeVersionBolsaB103ADMIN, error) {
	if ctx == nil || ctx.Err() != nil || poolGobierno == nil || poolCatalogo == nil ||
		poolGobierno == poolCatalogo || fuente == nil ||
		acreditarPoolCentral(ctx, poolGobierno, "vec_admin_version_rol_bolsa_b103_ejecutor") != nil ||
		acreditarPoolCentral(ctx, poolCatalogo, "vec_admin_catalogo_acciones_lector") != nil {
		return nil, errorArranque("version_bolsa_b103_pool")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("version_bolsa_b103_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("version_bolsa_b103_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	servicio, err := administracion.NuevoServicioVersionarRolBolsaB103V3(ctx,
		administracion.DependenciasVersionarRolBolsaV3{PoolGobierno: poolGobierno,
			FuenteCatalogo: fuente, Confianza: conf,
			DependenciasPDP: administracion.DependenciasConfianzaPerfilesV3{
				PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos,
				CatalogoMotivosID: base.CatalogoMotivosID, Firmante: firmante, Reloj: reloj,
				Generador:        seguridad.GeneradorReferenciasCriptograficas{},
				VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second},
			Motivos: c.Motivos, Reloj: reloj})
	if err != nil {
		return nil, errorArranque("version_bolsa_b103_servicio")
	}
	return &administracion.MontajeVersionBolsaB103ADMIN{Servicio: servicio, Fuente: fuente}, nil
}
