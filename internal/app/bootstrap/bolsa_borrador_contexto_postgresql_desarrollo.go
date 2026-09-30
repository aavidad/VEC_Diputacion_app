package bootstrap

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La referencia v1 se conserva para recuperar exactamente el registro histórico.
func operacionContextoBorradorBolsaDesarrollo() string {
	return referenciaAltaContratacionTemporalDesarrollo("oca_", "bolsa-bback:registro-contexto:v1")
}

func operacionContextoBorradorBolsaV2Desarrollo(contexto ports.ContextoAutorizacionAltaV3) (string, error) {
	if contexto.Resultado.Validar() != nil || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	// Sólo identidad ya acreditada: ni fecha, sesión, petición ni huella mutable.
	material := strings.Join([]string{
		"bolsa-bback:registro-contexto:v2", string(datos.Superficie), datos.CuentaRef,
		datos.PrincipalID, datos.PerfilActivoRef, datos.ContextoActorRef, datos.RegistroContextoRef,
	}, "\x00")
	return referenciaAltaContratacionTemporalDesarrollo("oca_", material), nil
}

type registroOperacionContextoBorradorBolsa struct {
	operacion, registro, cuenta, perfil string
}

// La selección sólo identifica la operación. El publicador común comprueba
// después toda la postimagen, incluidas procedencia, vigencias y punteros.
func elegirOperacionContextoBorradorBolsaDesarrollo(
	contexto ports.ContextoAutorizacionAltaV3,
	registros []registroOperacionContextoBorradorBolsa,
) (string, error) {
	scoped, err := operacionContextoBorradorBolsaV2Desarrollo(contexto)
	if err != nil {
		return "", err
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	legacy := operacionContextoBorradorBolsaDesarrollo()
	legacyPropia, scopedPropia := false, false
	for _, fila := range registros {
		mismaCuentaPerfil := fila.cuenta == datos.CuentaRef && fila.perfil == datos.PerfilActivoRef
		mismoRegistro := fila.registro == datos.RegistroContextoRef
		if fila.operacion == scoped {
			if !mismaCuentaPerfil || !mismoRegistro || scopedPropia {
				return "", falloPostgreSQLCTDesarrollo(nil)
			}
			scopedPropia = true
		}
		if mismoRegistro && (!mismaCuentaPerfil || (fila.operacion != legacy && fila.operacion != scoped)) {
			return "", falloPostgreSQLCTDesarrollo(nil)
		}
		if fila.operacion == legacy && mismaCuentaPerfil {
			if !mismoRegistro || legacyPropia {
				return "", falloPostgreSQLCTDesarrollo(nil)
			}
			legacyPropia = true
		}
	}
	if legacyPropia && scopedPropia {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	if legacyPropia {
		return legacy, nil
	}
	return scoped, nil
}

func seleccionarOperacionContextoBorradorBolsaDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, contexto ports.ContextoAutorizacionAltaV3,
) (string, error) {
	if ctx == nil || pool == nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	scoped, err := operacionContextoBorradorBolsaV2Desarrollo(contexto)
	if err != nil {
		return "", err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioContextoContratacionTemporalDesarrollo); err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	filas, err := tx.Query(ctx, `SELECT operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref
 FROM vec_contexto_actor_v1.registros_contexto
 WHERE operacion_ref IN ($1,$2) OR registro_contexto_ref=$3`,
		operacionContextoBorradorBolsaDesarrollo(), scoped, contexto.Resultado.RegistroContextoRef)
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	var registros []registroOperacionContextoBorradorBolsa
	for filas.Next() {
		var fila registroOperacionContextoBorradorBolsa
		if err := filas.Scan(&fila.operacion, &fila.registro, &fila.cuenta, &fila.perfil); err != nil {
			filas.Close()
			return "", falloPostgreSQLCTDesarrollo(err)
		}
		registros = append(registros, fila)
	}
	err = filas.Err()
	filas.Close()
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	operacion, err := elegirOperacionContextoBorradorBolsaDesarrollo(contexto, registros)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	return operacion, nil
}

// Registra el contexto nominal de Bolsa sin publicar una autorización V3.
func publicarContextoPostgreSQLBorradorBolsaDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteSesionBorradorBolsaDesarrollo,
) error {
	if soporte == nil || soporte.soporteCanal == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	contexto := soporte.soporteCanal.contexto
	operacion, err := seleccionarOperacionContextoBorradorBolsaDesarrollo(ctx, pool, contexto)
	if err != nil {
		return err
	}
	return publicarResultadoContextoPostgreSQLDesarrollo(ctx, pool, contexto.Resultado, operacion)
}
