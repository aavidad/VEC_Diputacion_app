package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const (
	capacidadProvisionar = "vec_identidad_sesiones_v1_provisionador"
	capacidadRegistrar   = "vec_identidad_sesiones_v1_registrador"
	capacidadRevalidar   = "vec_identidad_sesiones_v1_revalidador"
	capacidadRevocar     = "vec_identidad_sesiones_v1_revocador"
)

var capacidadesEjecucion = []string{
	capacidadProvisionar,
	capacidadRegistrar,
	capacidadRevalidar,
	capacidadRevocar,
}

const consultaAcreditarCapacidad = `
	SELECT session_user::text, current_user::text,
	       COALESCE(
	           login.rolcanlogin AND NOT login.rolsuper
	           AND NOT login.rolcreatedb AND NOT login.rolcreaterole
	           AND NOT login.rolreplication AND NOT login.rolbypassrls,
	           false
	       ),
	       COALESCE(
	           NOT grupo.rolcanlogin AND NOT grupo.rolsuper
	           AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
	           AND NOT grupo.rolreplication AND NOT grupo.rolbypassrls
	           AND membresia.inherit_option
	           AND NOT membresia.set_option
	           AND NOT membresia.admin_option,
	           false
	       ),
	       NOT EXISTS (
	           SELECT 1
	             FROM pg_catalog.pg_auth_members AS otra_membresia
	             JOIN pg_catalog.pg_roles AS otro_grupo
	               ON otro_grupo.oid = otra_membresia.roleid
	             JOIN pg_catalog.pg_roles AS otro_login
	               ON otro_login.oid = otra_membresia.member
	            WHERE otro_login.rolname = session_user
	              AND otro_grupo.rolname = ANY($2::text[])
	              AND otro_grupo.rolname <> $1
	       )
	  FROM pg_catalog.pg_roles AS login
	  JOIN pg_catalog.pg_roles AS grupo ON grupo.rolname = $1
	  JOIN pg_catalog.pg_auth_members AS membresia
	    ON membresia.roleid = grupo.oid
	   AND membresia.member = login.oid
	 WHERE login.rolname = session_user`

func acreditarCapacidadPool(
	ctx context.Context,
	pool *pgxpool.Pool,
	capacidadEsperada string,
) (string, error) {
	if ctx == nil || pool == nil {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	var usuarioSesion, usuarioActual string
	var loginSeguro, membresiaSegura, sinOtrasCapacidades bool
	err := pool.QueryRow(
		ctx,
		consultaAcreditarCapacidad,
		capacidadEsperada,
		capacidadesEjecucion,
	).Scan(
		&usuarioSesion,
		&usuarioActual,
		&loginSeguro,
		&membresiaSegura,
		&sinOtrasCapacidades,
	)
	if err != nil || usuarioSesion == "" || usuarioSesion != usuarioActual ||
		!loginSeguro || !membresiaSegura || !sinOtrasCapacidades {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	return usuarioSesion, nil
}
