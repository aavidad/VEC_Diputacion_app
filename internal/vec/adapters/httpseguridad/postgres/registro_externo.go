package postgres

import (
	"context"
	"crypto/rand"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const (
	consultaRegistrarSesionExterna = `
		SELECT autenticacion_ref, asercion_ref, sesion_ref,
		       control_sesion_ref, control_sesion_revision_texto,
		       control_sesion_estado, control_sesion_huella_sha256,
		       cuenta_ref, cuenta_ordinaria_ref,
		       sesion_revalidada_en, sesion_valida_hasta
		  FROM vec_identidad_externa_v1.registrar_sesion_v1(
		       $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		       $11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`
	consultaReconciliarSesionExterna = `
		SELECT autenticacion_ref, asercion_ref, sesion_ref,
		       control_sesion_ref, control_sesion_revision_texto,
		       control_sesion_estado, control_sesion_huella_sha256,
		       cuenta_ref, cuenta_ordinaria_ref,
		       sesion_revalidada_en, sesion_valida_hasta
		  FROM vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
		       $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		       $11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`
	consultaRevalidarSesionExterna = `
		SELECT vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
		       $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		       $11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`
)

// RegistroSesionesExternoPostgreSQL solo consume cuentas previamente
// provisionadas en la población externa. Sus dos LOGIN son exclusivos.
type RegistroSesionesExternoPostgreSQL struct {
	base *RegistroSesionesPostgreSQL
}

func NuevoRegistroSesionesExternoPostgreSQL(
	ctx context.Context,
	poolRegistro, poolRevalidacion *pgxpool.Pool,
	seudonimizador SeudonimizadorAlta,
	espacioIdentidad, dominioHMACRef string,
) (*RegistroSesionesExternoPostgreSQL, error) {
	if ctx == nil || poolRegistro == nil || poolRevalidacion == nil ||
		poolRegistro == poolRevalidacion || ctx.Err() != nil {
		return nil, httpseguridad.ErrRegistroSesionesAusente
	}
	usuarioRegistro, err := acreditarCapacidadExterna(
		ctx, poolRegistro, capacidadRegistrarExterna,
	)
	if err != nil {
		return nil, err
	}
	usuarioRevalidacion, err := acreditarCapacidadExterna(
		ctx, poolRevalidacion, capacidadRevalidarExterna,
	)
	if err != nil || usuarioRegistro == usuarioRevalidacion {
		return nil, httpseguridad.ErrRegistroSesionesAusente
	}
	return nuevoRegistroSesionesExternoPostgreSQL(
		poolRegistro, poolRevalidacion, seudonimizador,
		espacioIdentidad, dominioHMACRef, rand.Reader,
	)
}

func nuevoRegistroSesionesExternoPostgreSQL(
	registro, revalidacion iniciadorTransacciones,
	seudonimizador SeudonimizadorAlta,
	espacioIdentidad, dominioHMACRef string,
	aleatorio io.Reader,
) (*RegistroSesionesExternoPostgreSQL, error) {
	base, err := nuevoRegistroSesionesPostgreSQL(
		registro, revalidacion, seudonimizador,
		espacioIdentidad, dominioHMACRef, aleatorio,
	)
	if err != nil {
		return nil, err
	}
	return &RegistroSesionesExternoPostgreSQL{base: base}, nil
}

func (r *RegistroSesionesExternoPostgreSQL) ConsumirAsercionYRegistrar(
	ctx context.Context,
	alta httpseguridad.AltaSesionAtomica,
) (httpseguridad.ConfirmacionAltaSesion, error) {
	if r == nil || r.base == nil || valorNulo(ctx) || alta.Validar() != nil ||
		alta.Superficie != httpseguridad.SuperficieExternaPersonal ||
		alta.CuentaPrivilegiada || alta.CuentaOrdinariaID != "" {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, err
	}
	base := r.base
	if alta.EspacioIdentidad != base.espacioIdentidad {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	seudonimos, aliasOrdinario, err := SeudonimizarAltaConAliasCuentaOrdinaria(ctx, base.seudonimizador, IdentificadoresAlta{
		EspacioIdentidad: alta.EspacioIdentidad,
		AsercionID:       alta.AsercionID, SesionID: alta.SesionID,
		SujetoID: alta.SujetoID, CuentaID: alta.CuentaID,
	}, base.espacioIdentidad, base.dominioHMACRef)
	if err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, errorSesionSaneado(ctx)
	}
	operacionRef, err := nuevaReferenciaOperacion(base.aleatorio)
	if err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, errorSesionSaneado(ctx)
	}
	argumentos := argumentosAlta(operacionRef, seudonimos, aliasOrdinario, alta)
	respuesta, err := r.ejecutarAlta(ctx, argumentos)
	if err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, err
	}
	confirmacion := respuesta.confirmacion(alta)
	if confirmacion.ValidarPara(alta) != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	return confirmacion, nil
}

func (r *RegistroSesionesExternoPostgreSQL) ejecutarAlta(
	ctx context.Context,
	argumentos []any,
) (respuestaAlta, error) {
	tx, err := r.base.registro.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	defer revertir(tx)
	if prepararTransaccion(ctx, tx) != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	respuesta, err := consultarRespuestaAlta(ctx, tx, consultaRegistrarSesionExterna, argumentos)
	if err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	if tx.Commit(ctx) == nil {
		return respuesta, nil
	}
	// Un COMMIT incierto se coteja por su operación original, nunca se reenvía.
	ctxRecuperacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	txRecuperacion, err := r.base.registro.BeginTx(ctxRecuperacion, opcionesTransaccion())
	if err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	defer revertir(txRecuperacion)
	if prepararTransaccion(ctxRecuperacion, txRecuperacion) != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	recuperada, err := consultarRespuestaAlta(
		ctxRecuperacion, txRecuperacion, consultaReconciliarSesionExterna, argumentos,
	)
	if err != nil || !respuestasIguales(respuesta, recuperada) ||
		txRecuperacion.Commit(ctxRecuperacion) != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	return recuperada, nil
}

func (r *RegistroSesionesExternoPostgreSQL) ComprobarSesionYCuentaActivas(
	ctx context.Context,
	consulta httpseguridad.ConsultaSesionActiva,
) error {
	if r == nil || r.base == nil || valorNulo(ctx) || consulta.Validar() != nil ||
		consulta.Superficie != httpseguridad.SuperficieExternaPersonal ||
		consulta.CuentaPrivilegiada || consulta.CuentaRef != consulta.CuentaOrdinariaRef {
		return httpseguridad.ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := r.base.revalidacion.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return errorSesionSaneado(ctx)
	}
	defer revertir(tx)
	if prepararTransaccion(ctx, tx) != nil {
		return errorSesionSaneado(ctx)
	}
	var activa bool
	if tx.QueryRow(ctx, consultaRevalidarSesionExterna, argumentosConsulta(consulta)...).Scan(&activa) != nil ||
		!activa || tx.Commit(ctx) != nil {
		return errorSesionSaneado(ctx)
	}
	return nil
}

var _ httpseguridad.RegistroSesiones = (*RegistroSesionesExternoPostgreSQL)(nil)
