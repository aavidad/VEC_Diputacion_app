package bootstrap

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Preparación exclusiva del fixture de desarrollo, no autoridad corporativa.
// Conserva la cuenta del soporte creado por la composición de doble llave.
// No importa sus autenticaciones históricas ni crea sesiones o consumos.
// El pool de gobierno necesita SET ROLE al propietario de identidad únicamente
// en la base sintética; esta capacidad no corresponde al pool de peticiones.
func prepararCuentaNominalConsultasDesarrollo(
	ctx context.Context,
	gobierno *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	seudonimos postgresidentidad.SeudonimosAlta,
) error {
	if gobierno == nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	return prepararCuentaNominalConsultasDesarrolloConTransaccion(ctx, gobierno, soporte, seudonimos)
}

// Costura privada para probar el corte transaccional sin levantar PostgreSQL.
type transaccionesCuentaNominalDesarrollo interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func prepararCuentaNominalConsultasDesarrolloConTransaccion(
	ctx context.Context,
	gobierno transaccionesCuentaNominalDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	seudonimos postgresidentidad.SeudonimosAlta,
) error {
	cuenta, err := cuentaSinteticaNominalDesarrollo(ctx, soporte, seudonimos)
	if err != nil {
		return err
	}
	actoCuenta := referenciaAltaContratacionTemporalDesarrollo("opr_ct_dev_cuenta_", cuenta)
	actoEstado := referenciaAltaContratacionTemporalDesarrollo("opr_ct_dev_estado_", cuenta)
	actoActual := referenciaAltaContratacionTemporalDesarrollo("opr_ct_dev_actual_", cuenta)
	// El alias identifica cuenta/sujeto y generación. Aserción y sesión son
	// deliberadamente irrelevantes: los valores de arranque nunca se registran.
	actoAlias := referenciaAltaContratacionTemporalDesarrollo("opr_ct_dev_alias_",
		fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%x\x00%x",
			cuenta, seudonimos.EspacioIdentidad, seudonimos.Esquema,
			seudonimos.DominioRef, seudonimos.ClaveID, seudonimos.ClaveVersion,
			seudonimos.CuentaIDHMAC, seudonimos.SujetoIDHMAC))

	tx, err := gobierno.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	defer func() {
		limpieza, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(limpieza)
	}()
	if _, err = tx.Exec(ctx, configurarCuentaNominalDesarrolloSQL); err != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	var guardas, existe bool
	err = tx.QueryRow(ctx, consultarCuentaNominalDesarrolloSQL,
		cuenta, seudonimos.Esquema, seudonimos.DominioRef, seudonimos.ClaveID,
		int64(seudonimos.ClaveVersion), seudonimos.CuentaIDHMAC[:],
		seudonimos.SujetoIDHMAC[:]).Scan(&guardas, &existe)
	if err != nil || !guardas {
		return errorCuentaNominalDesarrollo(ctx)
	}
	if !existe {
		resultado, err := tx.Exec(ctx, incorporarCuentaNominalDesarrolloSQL,
			cuenta, actoCuenta, actoEstado, actoActual)
		if err != nil || resultado.RowsAffected() != 1 {
			return errorCuentaNominalDesarrollo(ctx)
		}
	}
	// También en replay: la API nominal admite un alias ya registrado antes
	// de mirar actividad. Aquí se bloquea y coteja el estado ANTES de llamarla.
	// Una cuenta parcial, inactiva o con otra revisión nunca se repara.
	var coincide bool
	err = tx.QueryRow(ctx, cotejarCuentaNominalDesarrolloSQL,
		cuenta, actoCuenta, actoEstado, actoActual).Scan(&coincide)
	if err != nil || !coincide {
		return errorCuentaNominalDesarrollo(ctx)
	}
	var cuentaAlias string
	err = tx.QueryRow(ctx, registrarAliasCuentaNominalDesarrolloSQL,
		actoAlias, cuenta, seudonimos.Esquema, seudonimos.DominioRef,
		seudonimos.ClaveID, int64(seudonimos.ClaveVersion),
		seudonimos.CuentaIDHMAC[:], seudonimos.SujetoIDHMAC[:]).Scan(&cuentaAlias)
	if err != nil || cuentaAlias != cuenta || ctx.Err() != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	return nil
}

func cuentaSinteticaNominalDesarrollo(
	ctx context.Context,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	seudonimos postgresidentidad.SeudonimosAlta,
) (string, error) {
	fallo := errorCuentaNominalDesarrollo(ctx)
	if contextoInterfazNulo(ctx) || ctx.Err() != nil || soporte == nil {
		return "", fallo
	}
	// No hay petición al arrancar: se verifica la configuración retenida por
	// el soporte sintético, no se fabrica una capacidad mTLS para este paso.
	soporte.mu.Lock()
	defer soporte.mu.Unlock()
	certificado, err := hex.DecodeString(soporte.certificadoSHA256)
	if soporte.sello == nil || !identificadorSesionDesarrolloValido(soporte.principalID) ||
		err != nil || len(certificado) != 32 ||
		hex.EncodeToString(certificado) != soporte.certificadoSHA256 ||
		soporte.contexto.Vinculo.ValidarPara(soporte.contexto.Resultado) != nil {
		return "", fallo
	}
	datos, err := soporte.contexto.Vinculo.Datos()
	base := soporte.principalID + "\x00" + soporte.certificadoSHA256
	cuenta := referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta")
	instantanea := soporte.contexto.Resultado.Contexto.Instantanea
	if err != nil || datos.CuentaPrivilegiada ||
		datos.CuentaRef != cuenta || datos.CuentaOrdinariaRef != cuenta ||
		datos.PrincipalID != referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona") ||
		datos.PerfilActivoRef != referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil") ||
		datos.ContextoActorRef != referenciaAltaContratacionTemporalDesarrollo("vca_", base+"\x00vinculo") ||
		datos.RegistroContextoRef != referenciaAltaContratacionTemporalDesarrollo("rca_", base+"\x00registro-contexto") ||
		datos.ContextoActorVersion != 1 || datos.ContextoActorCuentaVersion != 1 ||
		instantanea.PersonaVersion != 1 || instantanea.PerfilVersion != 1 ||
		instantanea.Estado != dominiovec.EstadoVinculoContextoActorActivo ||
		len(instantanea.Vinculos) != 0 ||
		datos.Superficie != dominiovec.SuperficieAutenticacionInternaCorporativaV1 ||
		datos.MetodoObservado != dominiovec.AuthMethodCertificate ||
		datos.GarantiaObservada != dominiovec.AuthAssuranceHigh {
		return "", fallo
	}
	// El llamador obtiene estos dos digests del seudonimizador de desarrollo:
	// CuentaID="desarrollo:"+datos.CuentaRef, SujetoID=soporte.principalID. No acepta namespace
	// corporativo ni otra clase de cuenta. No se verifican ni usan los digests
	// no operativos de aserción/sesión enviados por la fábrica al arrancar.
	if seudonimos.Esquema != postgresidentidad.EsquemaHMACSHA256V1 ||
		seudonimos.EspacioIdentidad != espacioIdentidadSesionDesarrollo ||
		seudonimos.DominioRef != dominioIdentidadSesionDesarrollo ||
		seudonimos.ClaveVersion == 0 || seudonimos.ClaveVersion > math.MaxInt64 ||
		seudonimos.ClaveID != fmt.Sprintf("vec.identidad.desarrollo.g%d", seudonimos.ClaveVersion) ||
		seudonimos.CuentaIDHMAC == ([32]byte{}) || seudonimos.SujetoIDHMAC == ([32]byte{}) ||
		seudonimos.CuentaIDHMAC == seudonimos.SujetoIDHMAC ||
		seudonimos.CuentaOrdinariaIDHMAC != ([32]byte{}) {
		return "", fallo
	}
	return cuenta, nil
}

func errorCuentaNominalDesarrollo(ctx context.Context) error {
	if !contextoInterfazNulo(ctx) && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrConsultaRRHHNoDisponible
}

const configurarCuentaNominalDesarrolloSQL = `
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '5s';
SET LOCAL idle_in_transaction_session_timeout = '10s';`

const consultarCuentaNominalDesarrolloSQL = `
SELECT (
    vec_identidad_sesiones_v1.referencia_valida($1::text, 'cta_') IS TRUE
    AND vec_identidad_sesiones_v1.coordenadas_hmac_validas(
        $2::text, $3::text, $4::text, $5::bigint) IS TRUE
    AND vec_identidad_sesiones_v1.huella_hmac_valida($6::bytea) IS TRUE
    AND vec_identidad_sesiones_v1.huella_hmac_valida($7::bytea) IS TRUE
    AND $6::bytea <> $7::bytea
), EXISTS (
    SELECT 1 FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref = $1::text
)`

const incorporarCuentaNominalDesarrolloSQL = `
WITH cuenta AS (
    INSERT INTO vec_identidad_sesiones_v1.cuenta (
        cuenta_ref, cuenta_privilegiada, cuenta_ordinaria_ref, provisionada_en, acto_ref
    ) VALUES ($1::text, false, NULL, clock_timestamp(), $2::text)
    RETURNING cuenta_ref, provisionada_en
), estado AS (
    INSERT INTO vec_identidad_sesiones_v1.estado_cuenta (
        cuenta_ref, revision, estado, registrada_en, acto_ref
    ) SELECT cuenta_ref, 1, 'activa', provisionada_en, $3::text FROM cuenta
    RETURNING cuenta_ref, revision, registrada_en
)
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual (
    cuenta_ref, revision, actualizada_en, acto_ref
) SELECT cuenta_ref, revision, registrada_en, $4::text FROM estado`

const cotejarCuentaNominalDesarrolloSQL = `
SELECT (
    NOT cuenta.cuenta_privilegiada AND cuenta.cuenta_ordinaria_ref IS NULL
    AND cuenta.acto_ref = $2::text
    AND actual.revision = 1 AND actual.acto_ref = $4::text
    AND estado.revision = 1 AND estado.estado = 'activa' AND estado.acto_ref = $3::text
    AND isfinite(cuenta.provisionada_en) AND cuenta.provisionada_en <= clock_timestamp()
    AND estado.registrada_en = cuenta.provisionada_en
    AND actual.actualizada_en = estado.registrada_en
    AND NOT EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.estado_cuenta AS historia
        WHERE historia.cuenta_ref = cuenta.cuenta_ref AND historia.revision <> 1
    )
)
FROM vec_identidad_sesiones_v1.cuenta AS cuenta
JOIN vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
  ON actual.cuenta_ref = cuenta.cuenta_ref
JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
  ON estado.cuenta_ref = actual.cuenta_ref AND estado.revision = actual.revision
WHERE cuenta.cuenta_ref = $1::text
FOR UPDATE OF actual`

const registrarAliasCuentaNominalDesarrolloSQL = `
SELECT vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
    $1::text, $2::text, $3::text, $4::text, $5::text, $6::bigint, $7::bytea, $8::bytea
)`
