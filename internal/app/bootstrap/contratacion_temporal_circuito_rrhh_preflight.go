package bootstrap

import (
	"context"
	"fmt"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// Comprueba las funciones y la ACL de lectura antes de montar la capacidad.
// Una configuración de circuito no habilita por sí misma una operación SQL.
func preflightCircuitoRRHHDesarrollo(base consultaPoliticaInformeNuevoCT) error {
	if dependenciaEsNulaContratacionTemporalDesarrollo(base) {
		return fmt.Errorf("circuito RRHH: clave=conexion_sql valor=false esperado=true")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	const consulta = `SELECT
	 pg_catalog.to_regprocedure('vec_contratacion_temporal.circuito_flujo_nuevo_ct164(jsonb)') IS NOT NULL,
	 coalesce(pg_catalog.has_function_privilege(current_user,
	  pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_circuito_rrhh_v1(text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
	  'EXECUTE'), false)`
	var actosInstalados, lecturaAutorizada bool
	if err := base.QueryRow(ctx, consulta).Scan(&actosInstalados, &lecturaAutorizada); err != nil {
		return fmt.Errorf("circuito RRHH: clave=comprobacion_sql valor=error esperado=ok")
	}
	if !actosInstalados {
		return fmt.Errorf("circuito RRHH: clave=ct164_instalada valor=false esperado=true")
	}
	if !lecturaAutorizada {
		return fmt.Errorf("circuito RRHH: clave=consulta_ct163_execute valor=false esperado=true")
	}
	return nil
}
