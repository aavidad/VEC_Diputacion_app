package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

const (
	funcionPrepararConfirmacionBorradorPostgreSQL = "vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2"
	funcionConfirmarBorradorPostgreSQL            = "vec_bolsa_convocatorias.confirmar_borrador_v1"
	duracionMinimaVentanaKMSPostgreSQL            = 100 * time.Millisecond
	duracionMaximaVentanaKMSPostgreSQL            = 5 * time.Second
)

var _ gobiernoconvocatorias.ConfirmadorAtomicoBorrador = (*ConfirmadorAtomicoBorradorPostgreSQL)(nil)

type esperadorConfirmacionBorradorPostgreSQL func(context.Context, time.Time) error

// ConfirmadorAtomicoBorradorPostgreSQL conserva A, revalidación KMS y B en
// una sola transacción SERIALIZABLE. No verifica el recibo poscommit: esa
// capacidad pertenece deliberadamente a otro pool y otra credencial.
type ConfirmadorAtomicoBorradorPostgreSQL struct {
	pool               iniciadorTransacciones
	revalidador        gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador
	identidad          gobiernoconvocatorias.IdentidadAutoridadBorrador
	vinculoVerificador gobiernoconvocatorias.VinculoVerificadorReciboBorrador
	esperar            esperadorConfirmacionBorradorPostgreSQL
}

func NuevoConfirmadorAtomicoBorradorPostgreSQL(
	pool *pgxpool.Pool,
	revalidador gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador,
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador,
	verificador *VerificadorReciboBorradorPostgreSQL,
) (*ConfirmadorAtomicoBorradorPostgreSQL, error) {
	return nuevoConfirmadorAtomicoBorradorPostgreSQL(
		pool, revalidador, identidad, verificador, esperarConfirmacionBorradorPostgreSQL,
	)
}

func nuevoConfirmadorAtomicoBorradorPostgreSQL(
	pool iniciadorTransacciones,
	revalidador gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador,
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador,
	verificador *VerificadorReciboBorradorPostgreSQL,
	esperar esperadorConfirmacionBorradorPostgreSQL,
) (*ConfirmadorAtomicoBorradorPostgreSQL, error) {
	descriptor, ok := revalidador.(gobiernoconvocatorias.DescriptorAutoridadBorrador)
	identidadVerificadorDB, identidadVerificadorCripto, verificadorValido :=
		identidadesVerificadorReciboPostgreSQL(verificador)
	identidadRevalidador := gobiernoconvocatorias.IdentidadAutoridadBorrador{}
	if ok && !valorNulo(revalidador) {
		identidadRevalidador = descriptor.IdentidadAutoridadBorrador()
	}
	if valorNulo(pool) || valorNulo(revalidador) || !ok || esperar == nil ||
		!identidadAutoridadBorradorPostgreSQLValida(identidad) ||
		!autoridadesBorradorPostgreSQLSeparadas(identidad, identidadRevalidador) ||
		!verificadorValido ||
		!autoridadesBorradorPostgreSQLSeparadas(identidad, identidadVerificadorDB) ||
		!autoridadesBorradorPostgreSQLSeparadas(identidad, identidadVerificadorCripto) ||
		!autoridadesBorradorPostgreSQLSeparadas(identidadRevalidador, identidadVerificadorDB) ||
		!autoridadesBorradorPostgreSQLSeparadas(identidadRevalidador, identidadVerificadorCripto) {
		return nil, gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	return &ConfirmadorAtomicoBorradorPostgreSQL{
		pool: pool, revalidador: revalidador, identidad: identidad,
		vinculoVerificador: verificador.VinculoVerificadorReciboBorrador(), esperar: esperar,
	}, nil
}

func (c *ConfirmadorAtomicoBorradorPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	if c == nil {
		return gobiernoconvocatorias.IdentidadAutoridadBorrador{}
	}
	return c.identidad
}

func (c *ConfirmadorAtomicoBorradorPostgreSQL) VinculoVerificadorReciboBorrador() gobiernoconvocatorias.VinculoVerificadorReciboBorrador {
	if c == nil {
		return gobiernoconvocatorias.VinculoVerificadorReciboBorrador{}
	}
	return c.vinculoVerificador
}

func (c *ConfirmadorAtomicoBorradorPostgreSQL) ConfirmarBorrador(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	if ctx == nil || c == nil || valorNulo(c.pool) || valorNulo(c.revalidador) || c.esperar == nil {
		return resultadoNoAplicadoBorradorPostgreSQL(), gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	if err := ctx.Err(); err != nil {
		return resultadoNoAplicadoBorradorPostgreSQL(), err
	}
	verificadorRef, err := c.vinculoVerificador.ReferenciaParaAcreditacion()
	if err != nil {
		return resultadoNoAplicadoBorradorPostgreSQL(),
			gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	carga, err := serializarConfirmacionBorradorPostgreSQL(solicitud)
	if err != nil {
		return resultadoNoAplicadoBorradorPostgreSQL(), gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	defer carga.borrar()
	tx, err := iniciarTransaccionBorradorPostgreSQL(ctx, c.pool, pgx.ReadWrite)
	if err != nil {
		return resultadoNoAplicadoBorradorPostgreSQL(), errorConfirmacionBorradorPostgreSQL(ctx, err)
	}
	fila, err := prepararConfirmacionBorradorPostgreSQL(ctx, tx, carga)
	defer fila.borrar()
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	if fila.resultado == "idempotencia_reutilizada" &&
		fila.estadoDiario == string(gobiernoconvocatorias.ResultadoDiarioConfirmado) {
		resultado, err := fila.restaurarConfirmada(solicitud)
		if err != nil {
			return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
		}
		return confirmarCommitBorradorPostgreSQL(ctx, tx, resultado)
	}
	if fila.resultado != "preparada" || !fila.requiereRevalidacion {
		resultado, err := fila.restaurarSinPreparacion()
		if err != nil {
			return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
		}
		return confirmarCommitBorradorPostgreSQL(ctx, tx, resultado)
	}
	provisional, huellaCuerpo, preparadaEn, err := fila.restaurarPreparada(solicitud)
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	limiteLocalVentanaKMS := time.Now().Add(provisional.ConfirmadaEn.Sub(preparadaEn))
	solicitudRevalidacion, err := gobiernoconvocatorias.NuevaSolicitudRevalidacionAtestacionKMSBorrador(
		solicitud, huellaCuerpo, preparadaEn,
	)
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	revalidacion, err := revalidarAtestacionKMSConPlazoPostgreSQL(
		ctx, provisional.ConfirmadaEn, c.revalidador, solicitudRevalidacion,
	)
	if err != nil || revalidacion.ValidarPara(solicitudRevalidacion) != nil ||
		revalidacion.ComprobadaEn.After(provisional.ConfirmadaEn) {
		if err == nil {
			err = gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	acreditacion, err := gobiernoconvocatorias.NuevaAcreditacionKMSConfirmacionBorrador(
		solicitud, solicitudRevalidacion, revalidacion, provisional,
		referenciaAcreditacionBorradorPostgreSQL(fila.preparacionRef.String, huellaCuerpo),
		1, verificadorRef,
	)
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	acreditacionJSON, err := serializarAcreditacionKMSBorradorPostgreSQL(acreditacion)
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	defer borrarBytesDiarioPostgreSQL(acreditacionJSON)
	notBeforeLocal := provisional.ConfirmadaEn
	if notBeforeLocal.After(limiteLocalVentanaKMS) {
		notBeforeLocal = limiteLocalVentanaKMS
	}
	if err := c.esperar(ctx, notBeforeLocal); err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	filaFinal, err := ejecutarFaseBConfirmacionBorradorPostgreSQL(
		ctx, tx, fila.preparacionRef.String, acreditacionJSON, fila.cuerpoCanonico,
	)
	defer filaFinal.borrar()
	if err != nil {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, err)
	}
	resultado, err := filaFinal.restaurarConfirmada(solicitud)
	if err != nil || resultado.AcreditacionKMS != acreditacion || resultado.Recibo != provisionalConAcreditacion(provisional, acreditacion) {
		return cerrarPorRollbackBorradorPostgreSQL(ctx, tx, gobiernoconvocatorias.ErrResultadoBorradorInseguro)
	}
	return confirmarCommitBorradorPostgreSQL(ctx, tx, resultado)
}

func ventanaRevalidacionKMSPostgreSQLValida(preparadaEn, confirmadaEn time.Time) bool {
	_, errPreparada := normalizarInstanteDiarioPostgreSQL(preparadaEn)
	_, errConfirmada := normalizarInstanteDiarioPostgreSQL(confirmadaEn)
	duracion := confirmadaEn.Sub(preparadaEn)
	return errPreparada == nil && errConfirmada == nil &&
		duracion >= duracionMinimaVentanaKMSPostgreSQL && duracion <= duracionMaximaVentanaKMSPostgreSQL
}

func revalidarAtestacionKMSConPlazoPostgreSQL(
	ctx context.Context,
	confirmadaEn time.Time,
	revalidador gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador,
	solicitud gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador,
) (gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador, error) {
	if ctx == nil || valorNulo(revalidador) || !ventanaRevalidacionKMSPostgreSQLValida(
		solicitud.SolicitadaEn, confirmadaEn,
	) {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	// Los instantes proceden del reloj autoritativo de PostgreSQL. Su diferencia
	// impone la cota dura incluso con desfase entre nodos; si el reloj local ve
	// un plazo menor, se conserva el más restrictivo.
	presupuesto := confirmadaEn.Sub(solicitud.SolicitadaEn)
	if restante := time.Until(confirmadaEn); restante <= 0 {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{},
			gobiernoconvocatorias.ErrOperacionBorradorEnCurso
	} else if restante < presupuesto {
		presupuesto = restante
	}
	ctxKMS, cancelar := context.WithTimeout(ctx, presupuesto)
	defer cancelar()
	resultado, err := revalidador.RevalidarAtestacionKMS(ctxKMS, solicitud)
	if err != nil {
		if ctx.Err() != nil {
			return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, ctx.Err()
		}
		if ctxKMS.Err() != nil {
			return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{},
				gobiernoconvocatorias.ErrOperacionBorradorEnCurso
		}
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if ctxKMS.Err() != nil {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{},
			gobiernoconvocatorias.ErrOperacionBorradorEnCurso
	}
	return resultado, nil
}

func esperarConfirmacionBorradorPostgreSQL(ctx context.Context, instante time.Time) error {
	duracion := time.Until(instante)
	if duracion <= 0 {
		return ctx.Err()
	}
	temporizador := time.NewTimer(duracion)
	defer temporizador.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-temporizador.C:
		return nil
	}
}

func referenciaAcreditacionBorradorPostgreSQL(preparacionRef, huellaCuerpo string) string {
	suma := sha256.Sum256([]byte("vec.bolsa.convocatoria.acreditacion-kms.v1\x00" + preparacionRef + "\x00" + huellaCuerpo))
	return "acreditacion-kms-borrador-" + hex.EncodeToString(suma[:])
}

func provisionalConAcreditacion(
	r gobiernoconvocatorias.ProyeccionReciboBorrador,
	a gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador,
) gobiernoconvocatorias.ProyeccionReciboBorrador {
	r.AcreditacionKMS = a
	return r
}

func resultadoNoAplicadoBorradorPostgreSQL() gobiernoconvocatorias.ResultadoConfirmacionAtomica {
	return gobiernoconvocatorias.ResultadoConfirmacionAtomica{Estado: gobiernoconvocatorias.ResultadoDiarioNoAplicado}
}

func resultadoIndeterminadoBorradorPostgreSQL() gobiernoconvocatorias.ResultadoConfirmacionAtomica {
	return gobiernoconvocatorias.ResultadoConfirmacionAtomica{Estado: gobiernoconvocatorias.ResultadoDiarioIndeterminado}
}

func iniciarTransaccionBorradorPostgreSQL(
	ctx context.Context, pool iniciadorTransacciones, modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: modo})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertir(tx)
		return nil, err
	}
	return tx, nil
}

func cerrarPorRollbackBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, causa error,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	ctxRollback, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelar()
	if err := tx.Rollback(ctxRollback); err != nil {
		return resultadoIndeterminadoBorradorPostgreSQL(), gobiernoconvocatorias.ErrOperacionBorradorIndeterminada
	}
	return resultadoNoAplicadoBorradorPostgreSQL(), errorConfirmacionBorradorPostgreSQL(ctx, causa)
}

func confirmarCommitBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, resultado gobiernoconvocatorias.ResultadoConfirmacionAtomica,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	if err := tx.Commit(ctx); err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			return resultadoNoAplicadoBorradorPostgreSQL(), gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
		return resultadoIndeterminadoBorradorPostgreSQL(), gobiernoconvocatorias.ErrOperacionBorradorIndeterminada
	}
	return resultado, nil
}

func errorConfirmacionBorradorPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) ||
		errors.Is(err, gobiernoconvocatorias.ErrRevalidacionKMSBorradorFallo) ||
		errors.Is(err, gobiernoconvocatorias.ErrCifradoBorradorInvalido) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso) {
		return gobiernoconvocatorias.ErrOperacionBorradorEnCurso
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) {
		switch errorPostgreSQL.Code {
		case "40001", "40P01", "55P03", "57014":
			return gobiernoconvocatorias.ErrOperacionBorradorEnCurso
		case "21000", "22000", "22003", "22023", "23503", "23505", "23514", "25000", "42501", "55000", "P0002":
			return gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
	}
	return gobiernoconvocatorias.ErrOperacionBorradorIndeterminada
}

func identidadAutoridadBorradorPostgreSQLValida(i gobiernoconvocatorias.IdentidadAutoridadBorrador) bool {
	reconstruida, err := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		i.ProveedorRef, i.InstanciaRef, i.CredencialRef, i.RolRef,
	)
	return err == nil && reconstruida == i
}

func autoridadesBorradorPostgreSQLSeparadas(a, b gobiernoconvocatorias.IdentidadAutoridadBorrador) bool {
	return identidadAutoridadBorradorPostgreSQLValida(a) && identidadAutoridadBorradorPostgreSQLValida(b) &&
		a != b && a.CredencialRef != b.CredencialRef && a.RolRef != b.RolRef &&
		(a.ProveedorRef != b.ProveedorRef || a.InstanciaRef != b.InstanciaRef)
}

func referenciaPostgreSQLValida(valor string) bool {
	identidad, err := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(valor, "instancia", "credencial", "rol")
	return err == nil && identidad.ProveedorRef == valor
}

func enteroPositivoConfirmacionPostgreSQL(valor pgtype.Int8) (uint64, error) {
	if !valor.Valid || valor.Int64 < 1 || valor.Int64 > math.MaxInt64 {
		return 0, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return uint64(valor.Int64), nil
}
