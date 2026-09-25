package interna

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var funcionesPersonalB2 = [...]string{
	"vec_personal.consultar_registro_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.consultar_vacantes_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.registrar_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.registrar_hecho_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.consultar_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
}

// acreditarPoolPersonalB2 revalida las siete funciones nominales sobre el
// LOGIN ya acreditado de Personal. No ejecuta una mutación ni instala SQL.
func acreditarPoolPersonalB2(ctx context.Context, pool *pgxpool.Pool, login string) error {
	if ctx == nil || ctx.Err() != nil || pool == nil || login == "" {
		return ErrPoolsSeguimientoNoDisponibles
	}
	perfil := perfilPoolSeguimiento{rol: "vec_personal_ejecutor", material: MaterialPoolSeguimiento{Login: login}}
	for _, funcion := range funcionesPersonalB2 {
		perfil.funcion = funcion
		if acreditarPoolSeguimiento(ctx, pool, perfil) != nil {
			return ErrPoolsSeguimientoNoDisponibles
		}
	}
	ctxSonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	const consulta = `
SELECT count(*)=11 AND NOT COALESCE(bool_or(pg_catalog.has_table_privilege(
  session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')),true)
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname='vec_personal' AND c.relname=ANY($1::text[]) AND c.relkind IN ('r','p')`
	tablas := []string{
		"relacion_servicio_historia", "ocupacion_empleado_historia",
		"servicio_reconocido_historia", "situacion_empleado_historia",
		"cobertura_ocupaciones_historia", "proyeccion_empleado_persona_historia",
		"recibo_lectura_registro_empleado_b2", "registro_empleado_b2_recibo",
		"entrada_catalogo_registro_empleado_historia", "entrada_catalogo_registro_empleado_actual",
		"recibo_lista_empleados_b2",
	}
	var sinDML bool
	if err := pool.QueryRow(ctxSonda, consulta, tablas).Scan(&sinDML); err != nil || !sinDML {
		return ErrPoolsSeguimientoNoDisponibles
	}
	return nil
}
