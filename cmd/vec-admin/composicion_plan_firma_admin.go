package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

// grupoGobiernoPlanFirma es el grupo dedicado de AD200: el pool del gobierno
// del plan debe ser un LOGIN exclusivo de él.
const grupoGobiernoPlanFirma = "vec_plan_firma_gobierno_ejecutor"

// componerGobiernoPlanFirmaADMIN monta el gobierno del plan nominal de firma
// sobre su pool propio y una cadena V3 con sólo la capacidad del gobierno. Los
// fallos se auditan con el registrador común y los motivos del overlay.
func componerGobiernoPlanFirmaADMIN(ctx context.Context, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	plan configuracionPlanFirmaPrivada, poolFuente, poolRegistro, poolMotivos, poolPlan *pgxpool.Pool,
	firmante ports.FirmanteAtestacionesAutorizacionV3, registrador ports.RegistradorIntentosAuditoria, reloj ports.Reloj,
) (*administracion.ServicioGobiernoPlanFirma, error) {
	if acreditarPoolCentral(ctx, poolPlan, grupoGobiernoPlanFirma) != nil {
		return nil, errorArranque("plan_firma_pool")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(plan.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("plan_firma_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("plan_firma_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	cadena, err := administracion.NuevaConfianzaGobiernoPlanFirmaV3(conf, administracion.DependenciasConfianzaPerfilesV3{
		PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos, CatalogoMotivosID: base.CatalogoMotivosID,
		Firmante: firmante, Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{},
		VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second})
	if err != nil {
		return nil, errorArranque("plan_firma_confianza_cadena")
	}
	emisor, err := administracion.NuevoEmisorGobiernoPlanFirma(cadena.Emisores[administracion.AudienciaGobiernoPlanFirmaV3], plan.Motivo, reloj)
	if err != nil {
		return nil, errorArranque("plan_firma_emisor")
	}
	autoridad, err := plannominal.NuevaAutoridadGobiernoPlanFirmaPostgreSQL(poolPlan, emisor, registrador,
		plannominal.ConfiguracionAuditoriaGobiernoPlanFirma{Proceso: u.Proceso, MotivoDenegado: u.MotivoDenegado,
			MotivoError: u.MotivoError, Plazo: u.plazoAuditoria()}, reloj)
	if err != nil {
		return nil, errorArranque("plan_firma_autoridad")
	}
	servicio, err := administracion.NuevoServicioGobiernoPlanFirma(autoridad)
	if err != nil {
		return nil, errorArranque("plan_firma_servicio")
	}
	return servicio, nil
}
