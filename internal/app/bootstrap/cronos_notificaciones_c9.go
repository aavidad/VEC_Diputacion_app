package bootstrap

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrCronosFaltaC9 = fmt.Errorf("%w: faltan consumidores nominales o funciones de notificaciones Cronos C9", ErrComposicionCronosEmpleadoNoDisponible)

type consultorCronosC9 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// El LOGIN de Cronos no necesita USAGE sobre autorización V3 para comprobar
// la fachada: la localiza por catálogo y verifica EXECUTE mediante su OID.
const preflightNotificacionesCronosC9 = `SELECT
  (SELECT count(*)=4 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname='vec_cronos_v1'
      AND p.proname IN ('registrar_notificacion_propia_v1','consultar_notificaciones_propio_v1',
        'consultar_bandeja_notificaciones_v1','atender_notificacion_v1')
      AND p.proargtypes='25 17 17 17 17 1700 1700 17 17 17 17'::oidvector
      AND p.prosecdef
      AND p.proowner='vec_cronos_v1_propietario'::regrole
      AND has_function_privilege('vec_cronos_v1_ejecutor',p.oid,'EXECUTE'))
  AND
  (SELECT count(*)=4 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname='vec_autorizacion_atestada_v3'
      AND p.proname IN ('registrar_y_consumir_cronos_notificacion_v3_atestada',
        'consumir_cronos_notificaciones_propio_v3_atestada',
        'consumir_cronos_notificaciones_bandeja_v3_atestada',
        'registrar_y_consumir_cronos_atencion_notificacion_v3_atestada')
      AND p.proargtypes='17 17 17 17 1700 1700 17 17 17 17'::oidvector
      AND p.prosecdef
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE'))
  AND to_regclass('vec_cronos_v1.notificacion') IS NOT NULL
  AND to_regclass('vec_cronos_v1.notificacion_atencion') IS NOT NULL`

func comprobarNotificacionesCronosC9(ctx context.Context, db consultorCronosC9) error {
	if ctx == nil || db == nil {
		return ErrCronosFaltaC9
	}
	var disponible bool
	if db.QueryRow(ctx, preflightNotificacionesCronosC9).Scan(&disponible) != nil || !disponible {
		return ErrCronosFaltaC9
	}
	return nil
}
