package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const (
	capacidadRegistrarExterna  = "vec_identidad_externa_v1_registrador"
	capacidadRevalidarExterna  = "vec_identidad_externa_v1_revalidador"
	esquemaAcreditacionInterno = "'vec_identidad_sesiones_v1'"
	esquemaAcreditacionExterno = "'vec_identidad_externa_v1'"
)

// El manifiesto y la consulta son cerrados. Ninguna firma, esquema o grupo
// procede de la petición ni de configuración de arranque.
var consultaAcreditarCapacidadExterna = strings.Replace(
	consultaAcreditarCapacidad,
	esquemaAcreditacionInterno,
	esquemaAcreditacionExterno,
	1,
)

func firmasCapacidadExterna(grupo string) ([]string, bool) {
	switch grupo {
	case capacidadRegistrarExterna:
		return []string{
			"vec_identidad_externa_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
			"vec_identidad_externa_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)",
		}, true
	case capacidadRevalidarExterna:
		return []string{
			"vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)",
			"vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(text,text)",
		}, true
	default:
		return nil, false
	}
}

func acreditarCapacidadExterna(
	ctx context.Context,
	pool *pgxpool.Pool,
	grupo string,
) (string, error) {
	if ctx == nil || pool == nil {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	return acreditarCapacidadExternaConsultor(ctx, pool, grupo)
}

func acreditarCapacidadExternaConsultor(
	ctx context.Context,
	consultor consultorFilaCapacidad,
	grupo string,
) (string, error) {
	if ctx == nil || valorNulo(consultor) || ctx.Err() != nil ||
		strings.Count(consultaAcreditarCapacidad, esquemaAcreditacionInterno) != 1 {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	firmas, valida := firmasCapacidadExterna(grupo)
	if !valida || len(firmas) != 2 {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	var usuarioSesion, usuarioActual string
	var loginSeguro, grupoSeguro, membresiaDirecta, membresiaExclusiva bool
	var firmasExactas, loginSinAutoridad, grupoExacto bool
	err := consultor.QueryRow(ctx, consultaAcreditarCapacidadExterna, grupo, firmas).Scan(
		&usuarioSesion, &usuarioActual,
		&loginSeguro, &grupoSeguro, &membresiaDirecta, &membresiaExclusiva,
		&firmasExactas, &loginSinAutoridad, &grupoExacto,
	)
	if err != nil || usuarioSesion == "" || usuarioSesion != usuarioActual ||
		!loginSeguro || !grupoSeguro || !membresiaDirecta ||
		!membresiaExclusiva || !firmasExactas || !loginSinAutoridad || !grupoExacto {
		return "", httpseguridad.ErrRegistroSesionesAusente
	}
	return usuarioSesion, nil
}
