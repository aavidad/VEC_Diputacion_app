package bootstrap

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// El lector acotado es una dependencia de la lista de candidatos. Una
// instalación incompleta deja esa lista no disponible sin degradarla a miles
// de consultas individuales; el cuadro y las demás rutas siguen montados.
func lectorBolsaRRHHConjuntoInstalado(ctx context.Context, pool *pgxpool.Pool) lectorBolsaRRHHConjunto {
	if ctx == nil || pool == nil {
		return nil
	}
	const preflight = `WITH firma AS (
		SELECT to_regprocedure('vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1(text,timestamptz)') AS oid
	) SELECT p.oid IS NOT NULL,coalesce(p.prosecdef,false),
		coalesce(p.proowner='vec_bolsa_llamamientos_propietario'::regrole,false),
		coalesce('search_path=pg_catalog, pg_temp'=ANY(p.proconfig),false),
		coalesce(has_function_privilege(current_user,p.oid,'EXECUTE'),false),
		coalesce((SELECT bool_or(a.grantee NOT IN (
			'vec_bolsa_llamamientos_propietario'::regrole,
			'vec_bolsa_llamamientos_ejecutor'::regrole))
			FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a),false),
		coalesce(has_function_privilege('vec_bolsa_llamamientos_relevo_cese',p.oid,'EXECUTE'),false)
		FROM firma LEFT JOIN pg_proc p ON p.oid=firma.oid`
	var existe, definidora, propietario, busqueda, ejecutor, permisoExtra, permisoRelevo bool
	if err := pool.QueryRow(ctx, preflight).Scan(&existe, &definidora, &propietario, &busqueda, &ejecutor, &permisoExtra, &permisoRelevo); err != nil {
		log.Printf("bolsa rrhh: clave=lector_b92_preflight esperado=firma_y_acl_cerradas actual=consulta_fallida causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
		return nil
	}
	if !existe || !definidora || !propietario || !busqueda || !ejecutor || permisoExtra || permisoRelevo {
		log.Printf("bolsa rrhh: clave=lector_b92_preflight esperado=funcion:definidora:propietario:busqueda:ejecutor:true_y_acl_extra:relevo:false actual=funcion:%t_definidora:%t_propietario:%t_busqueda:%t_ejecutor:%t_acl_extra:%t_relevo:%t",
			existe, definidora, propietario, busqueda, ejecutor, permisoExtra, permisoRelevo)
		return nil
	}
	lector, err := postgresbolsa.NuevoLectorBolsaRRHHConjunto(pool)
	if err != nil {
		log.Printf("bolsa rrhh: clave=lector_b92_construccion esperado=disponible actual=fallo causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
		return nil
	}
	return lector
}

// mapearSituacionesConjuntoBolsa ata el conjunto B82 a la instantánea y las
// entradas que el lector RRHH va a presentar. Una fila ausente, duplicada o
// de otra versión hace fallar la lectura completa.
func mapearSituacionesConjuntoBolsa(vigente ports.ConstitucionVigente, entradas []ports.EntradaConstitucion, filas []ports.SituacionBolsaRRHH) (map[string]ports.SituacionParticipacion, map[string]ports.EstadoCese, error) {
	if len(filas) != len(entradas) {
		return nil, nil, ErrComposicionDesarrolloIncompleta
	}
	esperadas := make(map[string]ports.EntradaConstitucion, len(entradas))
	for _, entrada := range entradas {
		if entrada.ParticipacionRef == "" || entrada.Orden == 0 {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if _, repetida := esperadas[entrada.ParticipacionRef]; repetida {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		esperadas[entrada.ParticipacionRef] = entrada
	}
	situaciones := make(map[string]ports.SituacionParticipacion, len(filas))
	ceses := make(map[string]ports.EstadoCese)
	for _, fila := range filas {
		entrada, existe := esperadas[fila.ParticipacionRef]
		if !existe || fila.BolsaRef != vigente.Bolsa.BolsaRef || fila.CategoriaRef != vigente.CategoriaRef ||
			!fila.ConfirmadaEn.Equal(vigente.ConfirmadaEn) || fila.InstantaneaRef != vigente.Instantanea.InstantaneaRef ||
			fila.VersionInstantanea != vigente.Instantanea.Version || fila.Orden != entrada.Orden ||
			fila.FilaNumero != entrada.FilaNumero || fila.Situacion == nil {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if _, repetida := situaciones[fila.ParticipacionRef]; repetida {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		situaciones[fila.ParticipacionRef] = *fila.Situacion
		if fila.Cese != nil {
			ceses[fila.ParticipacionRef] = *fila.Cese
		}
	}
	return situaciones, ceses, nil
}
