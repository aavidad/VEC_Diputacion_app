package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaConsultaRRHHPrueba struct {
	err      error
	destinos int
}

func (f *filaConsultaRRHHPrueba) Scan(destinos ...any) error {
	f.destinos = len(destinos)
	return f.err
}

type transaccionConsultaRRHHPrueba struct {
	fila           *filaConsultaRRHHPrueba
	consulta       string
	argumentos     int
	consultas      int
	confirmaciones int
	reversiones    int
	errCommit      error
}

func (t *transaccionConsultaRRHHPrueba) Begin(
	context.Context,
) (pgx.Tx, error) {
	return nil, errors.New("no usado")
}
func (t *transaccionConsultaRRHHPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return t.errCommit
}
func (t *transaccionConsultaRRHHPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}
func (*transaccionConsultaRRHHPrueba) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) SendBatch(
	context.Context,
	*pgx.Batch,
) pgx.BatchResults {
	return nil
}
func (*transaccionConsultaRRHHPrueba) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (*transaccionConsultaRRHHPrueba) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("no usado")
}
func (*transaccionConsultaRRHHPrueba) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, errors.New("no usado")
}
func (t *transaccionConsultaRRHHPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas++
	t.consulta = consulta
	t.argumentos = len(argumentos)
	return t.fila
}
func (*transaccionConsultaRRHHPrueba) Conn() *pgx.Conn { return nil }

type iniciadorConsultaRRHHPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	llamadas int
	err      error
}

func (i *iniciadorConsultaRRHHPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.llamadas++
	i.opciones = opciones
	return i.tx, i.err
}

func TestEjecutarConsultaRRHHFlujoNominalCuadroYDetalle(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre     string
		consulta   string
		argumentos int
		destinos   []any
	}{
		{
			nombre: "cuadro", consulta: consultaCuadroRRHHPostgreSQL,
			argumentos: 18,
			destinos: destinosCuadroConsultaRRHH(
				&salidaCuadroConsultaRRHH{},
			),
		},
		{
			nombre: "detalle", consulta: consultaDetalleRRHHPostgreSQL,
			argumentos: 15,
			destinos: destinosDetalleConsultaRRHH(
				&salidaDetalleConsultaRRHH{},
			),
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			fila := &filaConsultaRRHHPrueba{}
			tx := &transaccionConsultaRRHHPrueba{fila: fila}
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			validaciones := 0
			resultado, err := ejecutarConsultaRRHHEnTransaccion(
				context.Background(),
				iniciador,
				caso.consulta,
				make([]any, caso.argumentos),
				caso.destinos,
				func() (string, error) {
					validaciones++
					return "validado", nil
				},
			)
			if err != nil || resultado != "validado" {
				t.Fatalf("flujo nominal: %q, %v", resultado, err)
			}
			if iniciador.llamadas != 1 || tx.consultas != 1 ||
				tx.confirmaciones != 1 || tx.reversiones != 1 ||
				validaciones != 1 {
				t.Fatalf(
					"flujo/reintento inesperado: begin=%d query=%d "+
						"commit=%d rollback=%d validar=%d",
					iniciador.llamadas, tx.consultas, tx.confirmaciones,
					tx.reversiones, validaciones,
				)
			}
			if iniciador.opciones.IsoLevel != pgx.Serializable ||
				iniciador.opciones.AccessMode != pgx.ReadWrite ||
				tx.consulta != caso.consulta ||
				tx.argumentos != caso.argumentos ||
				fila.destinos != len(caso.destinos) {
				t.Fatalf("contrato transaccional alterado")
			}
		})
	}
}

func TestEjecutarConsultaRRHHFallaCerradoSinReintentos(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre       string
		errFila      error
		errValidar   error
		errCommit    error
		esperado     error
		commits      int
		validaciones int
	}{
		{
			nombre: "no_observable",
			errFila: &pgconn.PgError{
				Code: "42501", Message: "detalle privado",
			},
			esperado: ports.ErrConsultaRRHHNoObservable,
		},
		{
			nombre:       "analizador_rechaza",
			errValidar:   ports.ErrResultadoConsultaRRHHNoConfiable,
			esperado:     ports.ErrResultadoConsultaRRHHNoConfiable,
			validaciones: 1,
		},
		{
			nombre:    "commit_falla",
			errCommit: &pgconn.PgError{Code: "40001", Message: "privado"},
			esperado:  ports.ErrConsultaRRHHNoDisponible,
			commits:   1, validaciones: 1,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			fila := &filaConsultaRRHHPrueba{err: caso.errFila}
			tx := &transaccionConsultaRRHHPrueba{
				fila: fila, errCommit: caso.errCommit,
			}
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			validaciones := 0
			resultado, err := ejecutarConsultaRRHHEnTransaccion(
				context.Background(),
				iniciador,
				consultaCuadroRRHHPostgreSQL,
				make([]any, 18),
				destinosCuadroConsultaRRHH(&salidaCuadroConsultaRRHH{}),
				func() (string, error) {
					validaciones++
					return "no debe salir", caso.errValidar
				},
			)
			if resultado != "" || !errors.Is(err, caso.esperado) {
				t.Fatalf("resultado o error inseguro: %q, %v", resultado, err)
			}
			if iniciador.llamadas != 1 || tx.consultas != 1 ||
				tx.confirmaciones != caso.commits ||
				tx.reversiones != 1 ||
				validaciones != caso.validaciones {
				t.Fatalf(
					"hubo reintento/efecto: begin=%d query=%d "+
						"commit=%d rollback=%d validar=%d",
					iniciador.llamadas, tx.consultas, tx.confirmaciones,
					tx.reversiones, validaciones,
				)
			}
		})
	}
}
