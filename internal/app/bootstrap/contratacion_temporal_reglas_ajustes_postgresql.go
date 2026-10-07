package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
)

func prepararConsultaAjustesPostgreSQLCT(ctx context.Context, cfg config.Config, ejecucion *pgxpool.Pool,
	dependencias *dependenciasPostgreSQLContratacionTemporalDesarrollo,
) (bool, error) {
	motivo, activa, err := cargarMotivoAutorizacionAjustesCT(cfg)
	if err != nil || activa && !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas() {
		return false, errMotivoAutorizacionAjustes
	}
	if !activa {
		return false, nil
	}
	if dependencias == nil || preflightConsultaAjustesCT(ctx, ejecucion) != nil {
		return false, errMotivoAutorizacionAjustes
	}
	dependencias.motivoConsultaAjustesReglas = motivo
	dependencias.consultaAjustesReglasActiva = true
	return true, nil
}

func descriptoresMaterialAjustesCT(activa bool) []descriptorMaterialConsumidorV3Desarrollo {
	if !activa {
		return nil
	}
	return []descriptorMaterialConsumidorV3Desarrollo{descriptorMaterialConsultaAjustesCT()}
}

func proveedorMaterialConsultaAjustesCT(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo,
) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	return nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
		ctx, gobierno, material, reloj, catalogo, audienciaAjustesCT)
}
