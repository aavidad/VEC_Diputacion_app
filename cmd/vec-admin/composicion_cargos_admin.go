package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres/cargoadmin"
	"vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

// grupoCargosCompetenciales es el que exige Personal28 (y la rama de AD166)
// a la sesión: el pool de cargos debe ser un LOGIN exclusivo de él.
const grupoCargosCompetenciales = "vec_personal_ejecutor"

// componerCargosCompetencialesADMIN monta la publicación de cargos
// competenciales sobre su pool propio y una cadena V3 con sólo esa capacidad.
// Los fallos se auditan con el registrador común y los motivos del overlay.
func componerCargosCompetencialesADMIN(ctx context.Context, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	c configuracionCargosPrivada, poolFuente, poolRegistro, poolMotivos, poolCargos *pgxpool.Pool,
	firmante ports.FirmanteAtestacionesAutorizacionV3, registrador ports.RegistradorIntentosAuditoria, reloj ports.Reloj,
) (*administracion.ServicioEfectoNominal, error) {
	if acreditarPoolCentral(ctx, poolCargos, grupoCargosCompetenciales) != nil {
		return nil, errorArranque("cargos_pool")
	}
	if acreditarZonaHorariaUTC(ctx, poolCargos) != nil {
		return nil, errorArranque("cargos_zona_horaria")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("cargos_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("cargos_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	cadena, err := administracion.NuevaConfianzaEfectoNominalV3(conf, administracion.DependenciasConfianzaPerfilesV3{
		PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos, CatalogoMotivosID: base.CatalogoMotivosID,
		Firmante: firmante, Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{},
		VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second}, administracion.AudienciaCargoCompetencialV3)
	if err != nil {
		return nil, errorArranque("cargos_confianza_cadena")
	}
	contrato := personalpg.ContratoPublicacionCargoCompetencial()
	emisor, err := administracion.NuevoEmisorEfectoNominalADMIN(cadena.Emisores[administracion.AudienciaCargoCompetencialV3], c.Motivo, reloj, contrato)
	if err != nil {
		return nil, errorArranque("cargos_emisor")
	}
	ejecutor, err := efectonominaladmin.NuevoEjecutor(poolCargos, emisor, registrador, efectonominaladmin.ConfiguracionAuditoria{
		Proceso: u.Proceso, MotivoDenegado: u.MotivoDenegado, MotivoError: u.MotivoError, Plazo: u.plazoAuditoria()}, reloj, contrato)
	if err != nil {
		return nil, errorArranque("cargos_ejecutor")
	}
	servicio, err := administracion.NuevoServicioEfectoNominal(ejecutor)
	if err != nil {
		return nil, errorArranque("cargos_servicio")
	}
	return servicio, nil
}
