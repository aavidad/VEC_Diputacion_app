package postgres

import (
	"context"
	"encoding/json"
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
	funcionConsultarIdentidadesBorradorPostgreSQL = "vec_bolsa_convocatorias.consultar_identidades_borrador_v1"
	funcionReservarDecisionBorradorPostgreSQL     = "vec_bolsa_convocatorias.reservar_decision_borrador_v1"
	funcionReconciliarBorradorPostgreSQL          = "vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1"
	funcionReclamarReservaBorradorPostgreSQL      = "vec_bolsa_convocatorias.reclamar_reserva_borrador_v1"
)

var _ gobiernoconvocatorias.DiarioOperacionesBorrador = (*DiarioOperacionesBorradorPostgreSQL)(nil)

// DiarioOperacionesBorradorPostgreSQL solo invoca las funciones publicas
// SECURITY DEFINER del diario. La cuenta runtime no necesita acceso directo a
// tablas, secuencias ni funciones internas del esquema.
type DiarioOperacionesBorradorPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoDiarioOperacionesBorradorPostgreSQL(
	pool *pgxpool.Pool,
) (*DiarioOperacionesBorradorPostgreSQL, error) {
	return nuevoDiarioOperacionesBorradorPostgreSQL(pool)
}

func nuevoDiarioOperacionesBorradorPostgreSQL(
	pool iniciadorTransacciones,
) (*DiarioOperacionesBorradorPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	return &DiarioOperacionesBorradorPostgreSQL{pool: pool}, nil
}

func (d *DiarioOperacionesBorradorPostgreSQL) ConsultarIdentidades(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador,
) (gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador, error) {
	if err := validarContextoDiarioPostgreSQL(ctx, d); err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, err
	}
	identidades, err := serializarConsultaIdentidadesBorradorPostgreSQL(solicitud)
	if err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, err
	}
	defer borrarBytesDiarioPostgreSQL(identidades)
	tx, err := d.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, err
	}
	defer revertir(tx)

	fila, err := consultarFilaIdentidadesBorradorPostgreSQL(ctx, tx, identidades)
	// Scan puede haber escrito columnas anteriores antes de devolver un error.
	// Instalar la limpieza antes de clasificarlo evita conservar recibos o
	// identidades parcialmente restaurados en el heap del proceso.
	defer fila.borrar()
	if errors.Is(err, pgx.ErrNoRows) {
		resultado := gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}
		if resultado.ValidarPara(solicitud) != nil {
			return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{},
				gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
		if err := tx.Commit(ctx); err != nil {
			return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, errorDiarioPostgreSQL(ctx, err)
		}
		return resultado, nil
	}
	if err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	operacion, err := fila.operacion.restaurar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, err
	}
	resolucion, err := restaurarResolucionBorradorPostgreSQL(
		fila.identidadesConsultadas, fila.identidadPrimaria,
	)
	if err != nil || validarMetadatosConsultaBorradorPostgreSQL(fila, operacion) != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	resultado := gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{
		Coincidencias: []gobiernoconvocatorias.CoincidenciaIdentidadBorrador{{
			Resolucion: resolucion, Resultado: operacion,
		}},
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func (d *DiarioOperacionesBorradorPostgreSQL) ReservarDecision(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) (gobiernoconvocatorias.ResultadoReservaDecisionBorrador, error) {
	if err := validarContextoDiarioPostgreSQL(ctx, d); err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, err
	}
	carga, err := serializarReservaBorradorPostgreSQL(solicitud)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, err
	}
	defer carga.borrar()
	tx, err := d.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, err
	}
	defer revertir(tx)
	fila, err := reservarDecisionBorradorPostgreSQL(ctx, tx, carga)
	defer fila.borrar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	operacion, err := fila.operacion.restaurar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, err
	}
	resolucion, err := restaurarResolucionBorradorPostgreSQL(
		fila.identidadesConsultadas, fila.identidadPrimaria,
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, err
	}
	resultado := gobiernoconvocatorias.ResultadoReservaDecisionBorrador{
		Resolucion: resolucion, Resultado: operacion,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func (d *DiarioOperacionesBorradorPostgreSQL) Reconciliar(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudReconciliacionBorrador,
) (gobiernoconvocatorias.ResultadoReconciliacionBorrador, error) {
	if err := validarContextoDiarioPostgreSQL(ctx, d); err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, err
	}
	if solicitud.Validar() != nil || solicitud.Control.Revision > math.MaxInt64 ||
		solicitud.Control.Cercado > math.MaxInt64 {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{},
			gobiernoconvocatorias.ErrReconciliacionBorradorInvalida
	}
	identidad, err := serializarIdentidadBorradorPostgreSQL(solicitud.IdentidadPrimaria)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, err
	}
	defer borrarBytesDiarioPostgreSQL(identidad)
	tx, err := d.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, err
	}
	defer revertir(tx)
	fila, err := reconciliarBorradorPostgreSQL(ctx, tx, identidad, solicitud)
	defer fila.operacion.borrar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	operacion, err := fila.operacion.restaurar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, err
	}
	comprobada, err := restaurarInstanteObligatorioDiarioPostgreSQL(fila.comprobadaEn)
	if err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, err
	}
	resultado := gobiernoconvocatorias.ResultadoReconciliacionBorrador{
		Resultado: operacion, ComprobadaEn: comprobada,
		PruebaDesenlaceRef: textoNuloDiarioPostgreSQL(fila.pruebaDesenlaceRef),
		HuellaPruebaSHA256: textoNuloDiarioPostgreSQL(fila.huellaPruebaDesenlace),
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, errorDiarioPostgreSQL(ctx, err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return gobiernoconvocatorias.ResultadoReconciliacionBorrador{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func (d *DiarioOperacionesBorradorPostgreSQL) ReclamarDecision(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador,
) (gobiernoconvocatorias.ResultadoOperacionDiario, error) {
	if err := validarContextoDiarioPostgreSQL(ctx, d); err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, err
	}
	if solicitud.Validar() != nil || solicitud.Reconciliacion.Resultado.Revision > math.MaxInt64 ||
		solicitud.Reconciliacion.Resultado.Cercado > math.MaxInt64 {
		return gobiernoconvocatorias.ResultadoOperacionDiario{},
			gobiernoconvocatorias.ErrReclamacionBorradorInvalida
	}
	carga, err := serializarReservaValidadaBorradorPostgreSQL(solicitud.Nueva)
	if err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, err
	}
	defer carga.borrar()
	tx, err := d.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, err
	}
	defer revertir(tx)
	fila, err := reclamarReservaBorradorPostgreSQL(ctx, tx, carga, solicitud)
	defer fila.borrar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, errorDiarioPostgreSQL(ctx, err)
	}
	operacion, err := fila.operacion.restaurar()
	if err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, err
	}
	identidad, err := restaurarIdentidadDesdeJSONDiarioPostgreSQL(fila.identidad)
	if err != nil || !identidadesDiarioPostgreSQLIguales(identidad, solicitud.Nueva.Proyeccion.IdentidadPrimaria) ||
		validarReclamacionRestauradaDiarioPostgreSQL(solicitud, operacion) != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, errorDiarioPostgreSQL(ctx, err)
	}
	if validarReclamacionRestauradaDiarioPostgreSQL(solicitud, operacion) != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{},
			gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return operacion, nil
}

func (d *DiarioOperacionesBorradorPostgreSQL) iniciar(
	ctx context.Context,
	modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: modo})
	if err != nil {
		return nil, errorDiarioPostgreSQL(ctx, err)
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
		return nil, errorDiarioPostgreSQL(ctx, err)
	}
	return tx, nil
}

type filaOperacionBorradorPostgreSQL struct {
	estado                string
	revision              pgtype.Int8
	cercado               pgtype.Int8
	arrendamientoIniciaEn pgtype.Timestamptz
	arrendamientoVenceEn  pgtype.Timestamptz
	recibo                []byte
}

func (f *filaOperacionBorradorPostgreSQL) destinos() []any {
	return []any{&f.estado, &f.revision, &f.cercado, &f.arrendamientoIniciaEn, &f.arrendamientoVenceEn, &f.recibo}
}

func (f *filaOperacionBorradorPostgreSQL) borrar() { borrarBytesDiarioPostgreSQL(f.recibo) }

func (f filaOperacionBorradorPostgreSQL) restaurar() (gobiernoconvocatorias.ResultadoOperacionDiario, error) {
	estado := gobiernoconvocatorias.EstadoResultadoDiario(f.estado)
	switch estado {
	case gobiernoconvocatorias.ResultadoDiarioAusente,
		gobiernoconvocatorias.ResultadoDiarioReservado,
		gobiernoconvocatorias.ResultadoDiarioEnCurso,
		gobiernoconvocatorias.ResultadoDiarioIndeterminado,
		gobiernoconvocatorias.ResultadoDiarioConfirmado,
		gobiernoconvocatorias.ResultadoDiarioNoAplicado,
		gobiernoconvocatorias.ResultadoDiarioConflicto:
	default:
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	revision, errRevision := enteroNoNegativoDiarioPostgreSQL(f.revision)
	cercado, errCercado := enteroNoNegativoDiarioPostgreSQL(f.cercado)
	inicio, errInicio := restaurarInstanteNuloDiarioPostgreSQL(f.arrendamientoIniciaEn)
	vence, errVence := restaurarInstanteNuloDiarioPostgreSQL(f.arrendamientoVenceEn)
	recibo, errRecibo := restaurarReciboBorradorPostgreSQL(f.recibo)
	if errors.Join(errRevision, errCercado, errInicio, errVence, errRecibo) != nil {
		return gobiernoconvocatorias.ResultadoOperacionDiario{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return gobiernoconvocatorias.ResultadoOperacionDiario{
		Estado: estado, Revision: revision, Cercado: cercado,
		ArrendamientoIniciaEn: inicio, ArrendamientoVenceEn: vence, Recibo: recibo,
	}, nil
}

type filaConsultaIdentidadesBorradorPostgreSQL struct {
	operacion               filaOperacionBorradorPostgreSQL
	transaccionRef          pgtype.Text
	accion                  pgtype.Text
	estadoPrincipalRef      pgtype.Text
	estadoPrincipalRevision pgtype.Int8
	estadoPrincipalHuella   pgtype.Text
	auditoriaRef            pgtype.Text
	huellaAuditoria         pgtype.Text
	eventoOutboxRef         pgtype.Text
	huellaEventoOutbox      pgtype.Text
	confirmadaEn            pgtype.Timestamptz
	identidadesConsultadas  []byte
	identidadPrimaria       []byte
}

func (f *filaConsultaIdentidadesBorradorPostgreSQL) borrar() {
	f.operacion.borrar()
	borrarBytesDiarioPostgreSQL(f.identidadesConsultadas, f.identidadPrimaria)
}

func consultarFilaIdentidadesBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, identidades []byte,
) (filaConsultaIdentidadesBorradorPostgreSQL, error) {
	var f filaConsultaIdentidadesBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT estado, revision, cercado, arrendamiento_inicia_en,
		       arrendamiento_vence_en, transaccion_ref, accion,
		       estado_principal_ref, estado_principal_revision,
		       estado_principal_huella_sha256, auditoria_ref,
		       huella_auditoria_sha256, evento_outbox_ref,
		       huella_evento_outbox_sha256, confirmada_en, recibo,
		       identidades_consultadas, identidad_primaria
		FROM `+funcionConsultarIdentidadesBorradorPostgreSQL+`($1::jsonb)`, identidades,
	).Scan(
		&f.operacion.estado, &f.operacion.revision, &f.operacion.cercado,
		&f.operacion.arrendamientoIniciaEn, &f.operacion.arrendamientoVenceEn,
		&f.transaccionRef, &f.accion, &f.estadoPrincipalRef, &f.estadoPrincipalRevision,
		&f.estadoPrincipalHuella, &f.auditoriaRef, &f.huellaAuditoria,
		&f.eventoOutboxRef, &f.huellaEventoOutbox, &f.confirmadaEn, &f.operacion.recibo,
		&f.identidadesConsultadas, &f.identidadPrimaria,
	)
	return f, err
}

type filaReservaBorradorPostgreSQL struct {
	operacion              filaOperacionBorradorPostgreSQL
	identidadesConsultadas []byte
	identidadPrimaria      []byte
}

func (f *filaReservaBorradorPostgreSQL) borrar() {
	f.operacion.borrar()
	borrarBytesDiarioPostgreSQL(f.identidadesConsultadas, f.identidadPrimaria)
}

func reservarDecisionBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, carga cargaReservaBorradorPostgreSQL,
) (filaReservaBorradorPostgreSQL, error) {
	var f filaReservaBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT estado, revision, cercado, arrendamiento_inicia_en,
		       arrendamiento_vence_en, recibo, identidades_consultadas,
		       identidad_primaria
		FROM `+funcionReservarDecisionBorradorPostgreSQL+`(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea, $5::bytea, $6::bytea
		)`, carga.Reserva, carga.Prueba, carga.Material, carga.Version, carga.Decision, carga.Contexto,
	).Scan(
		&f.operacion.estado, &f.operacion.revision, &f.operacion.cercado,
		&f.operacion.arrendamientoIniciaEn, &f.operacion.arrendamientoVenceEn,
		&f.operacion.recibo, &f.identidadesConsultadas, &f.identidadPrimaria,
	)
	return f, err
}

type filaReconciliacionBorradorPostgreSQL struct {
	operacion             filaOperacionBorradorPostgreSQL
	pruebaDesenlaceRef    pgtype.Text
	huellaPruebaDesenlace pgtype.Text
	comprobadaEn          pgtype.Timestamptz
}

func reconciliarBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, identidad []byte,
	solicitud gobiernoconvocatorias.SolicitudReconciliacionBorrador,
) (filaReconciliacionBorradorPostgreSQL, error) {
	var f filaReconciliacionBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT estado, revision, cercado, arrendamiento_inicia_en,
		       arrendamiento_vence_en, prueba_desenlace_ref,
		       huella_prueba_desenlace_sha256, comprobada_en, recibo
		FROM `+funcionReconciliarBorradorPostgreSQL+`(
			$1::jsonb, $2::text, $3::bigint, $4::bigint, $5::timestamptz
		)`, identidad, string(solicitud.Control.Estado), int64(solicitud.Control.Revision),
		int64(solicitud.Control.Cercado), solicitud.SolicitadaEn,
	).Scan(
		&f.operacion.estado, &f.operacion.revision, &f.operacion.cercado,
		&f.operacion.arrendamientoIniciaEn, &f.operacion.arrendamientoVenceEn,
		&f.pruebaDesenlaceRef, &f.huellaPruebaDesenlace, &f.comprobadaEn, &f.operacion.recibo,
	)
	return f, err
}

type filaReclamacionBorradorPostgreSQL struct {
	operacion filaOperacionBorradorPostgreSQL
	identidad []byte
}

func (f *filaReclamacionBorradorPostgreSQL) borrar() {
	f.operacion.borrar()
	borrarBytesDiarioPostgreSQL(f.identidad)
}

func reclamarReservaBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, carga cargaReservaBorradorPostgreSQL,
	solicitud gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador,
) (filaReclamacionBorradorPostgreSQL, error) {
	var f filaReclamacionBorradorPostgreSQL
	anterior := solicitud.Reconciliacion.Resultado
	err := tx.QueryRow(ctx, `
		SELECT estado, revision, cercado, arrendamiento_inicia_en,
		       arrendamiento_vence_en, recibo, identidad
		FROM `+funcionReclamarReservaBorradorPostgreSQL+`(
			$1::bigint, $2::bigint, $3::jsonb, $4::jsonb, $5::bytea,
			$6::bytea, $7::bytea, $8::bytea
		)`, int64(anterior.Revision), int64(anterior.Cercado), carga.Reserva, carga.Prueba,
		carga.Material, carga.Version, carga.Decision, carga.Contexto,
	).Scan(
		&f.operacion.estado, &f.operacion.revision, &f.operacion.cercado,
		&f.operacion.arrendamientoIniciaEn, &f.operacion.arrendamientoVenceEn,
		&f.operacion.recibo, &f.identidad,
	)
	return f, err
}

func restaurarResolucionBorradorPostgreSQL(
	identidadesJSON, primariaJSON []byte,
) (gobiernoconvocatorias.ResolucionIdentidadBorrador, error) {
	var persistidas []identidadDiarioPostgreSQL
	if err := decodificarJSONCerradoDiarioPostgreSQL(identidadesJSON, &persistidas); err != nil || len(persistidas) == 0 {
		return gobiernoconvocatorias.ResolucionIdentidadBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	identidades := make([]gobiernoconvocatorias.ProyeccionIdentidadOperacion, len(persistidas))
	for indice, persistida := range persistidas {
		identidad, err := restaurarIdentidadDiarioPostgreSQL(persistida)
		if err != nil {
			return gobiernoconvocatorias.ResolucionIdentidadBorrador{}, err
		}
		identidades[indice] = identidad
	}
	primaria, err := restaurarIdentidadDesdeJSONDiarioPostgreSQL(primariaJSON)
	if err != nil {
		return gobiernoconvocatorias.ResolucionIdentidadBorrador{}, err
	}
	return gobiernoconvocatorias.ResolucionIdentidadBorrador{
		IdentidadesConsultadas: identidades, IdentidadPrimaria: primaria,
	}, nil
}

func serializarIdentidadBorradorPostgreSQL(
	identidad gobiernoconvocatorias.ProyeccionIdentidadOperacion,
) ([]byte, error) {
	proyeccion, err := proyectarIdentidadDiarioPostgreSQL(identidad)
	if err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(proyeccion)
	if err != nil {
		return nil, gobiernoconvocatorias.ErrReservaBorradorInvalida
	}
	return contenido, nil
}

func restaurarIdentidadDesdeJSONDiarioPostgreSQL(
	contenido []byte,
) (gobiernoconvocatorias.ProyeccionIdentidadOperacion, error) {
	var persistida identidadDiarioPostgreSQL
	if err := decodificarJSONCerradoDiarioPostgreSQL(contenido, &persistida); err != nil {
		return gobiernoconvocatorias.ProyeccionIdentidadOperacion{}, err
	}
	return restaurarIdentidadDiarioPostgreSQL(persistida)
}

func validarMetadatosConsultaBorradorPostgreSQL(
	fila filaConsultaIdentidadesBorradorPostgreSQL,
	operacion gobiernoconvocatorias.ResultadoOperacionDiario,
) error {
	if operacion.Estado != gobiernoconvocatorias.ResultadoDiarioConfirmado {
		return nil
	}
	if operacion.Recibo == nil || !fila.transaccionRef.Valid || !fila.accion.Valid ||
		!fila.estadoPrincipalRef.Valid || !fila.estadoPrincipalRevision.Valid ||
		!fila.estadoPrincipalHuella.Valid || !fila.auditoriaRef.Valid || !fila.huellaAuditoria.Valid ||
		!fila.eventoOutboxRef.Valid || !fila.huellaEventoOutbox.Valid || !fila.confirmadaEn.Valid {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	r := operacion.Recibo
	confirmada, err := restaurarInstanteObligatorioDiarioPostgreSQL(fila.confirmadaEn)
	if err != nil || fila.estadoPrincipalRevision.Int64 < 1 || fila.estadoPrincipalRevision.Int64 > math.MaxInt ||
		fila.transaccionRef.String != r.TransaccionRef || fila.accion.String != r.Accion ||
		fila.estadoPrincipalRef.String != r.EstadoPrincipal.Referencia ||
		int(fila.estadoPrincipalRevision.Int64) != r.EstadoPrincipal.Revision ||
		fila.estadoPrincipalHuella.String != r.EstadoPrincipal.HuellaEstadoSHA256 ||
		fila.auditoriaRef.String != r.AuditoriaRef || fila.huellaAuditoria.String != r.HuellaAuditoriaSHA256 ||
		fila.eventoOutboxRef.String != r.EventoOutboxRef ||
		fila.huellaEventoOutbox.String != r.HuellaEventoOutboxSHA256 || !confirmada.Equal(r.ConfirmadaEn) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func validarReclamacionRestauradaDiarioPostgreSQL(
	solicitud gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador,
	resultado gobiernoconvocatorias.ResultadoOperacionDiario,
) error {
	anterior := solicitud.Reconciliacion.Resultado
	proyeccion := solicitud.Nueva.Proyeccion
	if resultado.Estado != gobiernoconvocatorias.ResultadoDiarioReservado || resultado.Recibo != nil ||
		resultado.Revision <= anterior.Revision || resultado.Cercado <= anterior.Cercado ||
		!resultado.ArrendamientoIniciaEn.Equal(proyeccion.ArrendamientoIniciaEn) ||
		!resultado.ArrendamientoVenceEn.Equal(proyeccion.ArrendamientoVenceEn) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func validarContextoDiarioPostgreSQL(ctx context.Context, diario *DiarioOperacionesBorradorPostgreSQL) error {
	if ctx == nil || diario == nil || valorNulo(diario.pool) {
		return gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func enteroNoNegativoDiarioPostgreSQL(valor pgtype.Int8) (uint64, error) {
	if !valor.Valid {
		return 0, nil
	}
	if valor.Int64 < 0 {
		return 0, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return uint64(valor.Int64), nil
}

func restaurarInstanteNuloDiarioPostgreSQL(valor pgtype.Timestamptz) (time.Time, error) {
	if !valor.Valid {
		return time.Time{}, nil
	}
	return normalizarInstanteDiarioPostgreSQL(valor.Time)
}

func restaurarInstanteObligatorioDiarioPostgreSQL(valor pgtype.Timestamptz) (time.Time, error) {
	if !valor.Valid {
		return time.Time{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return normalizarInstanteDiarioPostgreSQL(valor.Time)
}

func normalizarInstanteDiarioPostgreSQL(valor time.Time) (time.Time, error) {
	if valor.IsZero() || valor.Nanosecond()%1_000 != 0 {
		return time.Time{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return valor.UTC(), nil
}

func textoNuloDiarioPostgreSQL(valor pgtype.Text) string {
	if !valor.Valid {
		return ""
	}
	return valor.String
}

func identidadesDiarioPostgreSQLIguales(
	a, b gobiernoconvocatorias.ProyeccionIdentidadOperacion,
) bool {
	return a.Localizador.VersionEsquema == b.Localizador.VersionEsquema &&
		a.Localizador.Dominio == b.Localizador.Dominio && a.Localizador.ClaveRef == b.Localizador.ClaveRef &&
		a.Localizador.GeneracionClave == b.Localizador.GeneracionClave &&
		a.Localizador.ValorHMACSHA256 == b.Localizador.ValorHMACSHA256 &&
		a.HuellaSolicitud.VersionEsquema == b.HuellaSolicitud.VersionEsquema &&
		a.HuellaSolicitud.Dominio == b.HuellaSolicitud.Dominio &&
		a.HuellaSolicitud.ClaveRef == b.HuellaSolicitud.ClaveRef &&
		a.HuellaSolicitud.GeneracionClave == b.HuellaSolicitud.GeneracionClave &&
		a.HuellaSolicitud.ValorHMACSHA256 == b.HuellaSolicitud.ValorHMACSHA256
}

func errorDiarioPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) {
		switch errorPostgreSQL.Code {
		case "21000":
			return gobiernoconvocatorias.ErrConsultaIdempotenciaAmbigua
		case "40001", "40P01", "55P03", "57014":
			return gobiernoconvocatorias.ErrOperacionBorradorEnCurso
		case "22000", "22003", "22023", "23503", "23505", "23514", "42501", "55000", "P0002":
			return gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
	}
	return gobiernoconvocatorias.ErrOperacionBorradorIndeterminada
}
