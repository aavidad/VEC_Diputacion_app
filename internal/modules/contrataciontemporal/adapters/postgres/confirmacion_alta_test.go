package postgres

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type iniciadorConfirmacionPrueba struct {
	mu            sync.Mutex
	transacciones []pgx.Tx
	errores       []error
	opciones      []pgx.TxOptions
	inicios       int
	fabrica       func() pgx.Tx
}

func (i *iniciadorConfirmacionPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	indice := i.inicios
	i.inicios++
	i.opciones = append(i.opciones, opciones)
	if indice < len(i.errores) && i.errores[indice] != nil {
		return nil, i.errores[indice]
	}
	if indice < len(i.transacciones) {
		return i.transacciones[indice], nil
	}
	if i.fabrica != nil {
		return i.fabrica(), nil
	}
	return nil, errors.New("transacción de prueba no disponible")
}

type transaccionConfirmacionPrueba struct {
	pgx.Tx
	filas          pgx.Rows
	errConsulta    error
	errConfigurar  error
	errConfirmar   error
	alConsultar    func()
	alConfirmar    func()
	consulta       string
	configuracion  string
	argumentos     []any
	confirmaciones int
	reversiones    int
}

type errorReintentoSeguroPrueba struct{}

func (errorReintentoSeguroPrueba) Error() string     { return "fallo previo al envío" }
func (errorReintentoSeguroPrueba) SafeToRetry() bool { return true }

func (t *transaccionConfirmacionPrueba) Exec(
	_ context.Context,
	sql string,
	_ ...any,
) (pgconn.CommandTag, error) {
	t.configuracion = sql
	return pgconn.NewCommandTag("SELECT 1"), t.errConfigurar
}

func (t *transaccionConfirmacionPrueba) Query(
	_ context.Context,
	sql string,
	argumentos ...any,
) (pgx.Rows, error) {
	t.consulta = sql
	t.argumentos = clonarArgumentosConfirmacion(argumentos)
	if t.alConsultar != nil {
		t.alConsultar()
	}
	return t.filas, t.errConsulta
}

func (t *transaccionConfirmacionPrueba) Commit(context.Context) error {
	t.confirmaciones++
	if t.alConfirmar != nil {
		t.alConfirmar()
	}
	return t.errConfirmar
}

func (t *transaccionConfirmacionPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filasConfirmacionPrueba struct {
	pgx.Rows
	valores [][]any
	actual  int
	err     error
}

func (f *filasConfirmacionPrueba) Close() {}

func (f *filasConfirmacionPrueba) Err() error { return f.err }

func (f *filasConfirmacionPrueba) Next() bool {
	if f.actual >= len(f.valores) {
		return false
	}
	f.actual++
	return true
}

func (f *filasConfirmacionPrueba) Scan(destinos ...any) error {
	if f.actual == 0 || f.actual > len(f.valores) {
		return errors.New("fila de prueba fuera de rango")
	}
	valores := f.valores[f.actual-1]
	if len(destinos) != len(valores) {
		return errors.New("columnas inesperadas")
	}
	for indice, destino := range destinos {
		switch puntero := destino.(type) {
		case *string:
			valor, ok := valores[indice].(string)
			if !ok {
				return errors.New("texto inválido")
			}
			*puntero = valor
		case *time.Time:
			valor, ok := valores[indice].(time.Time)
			if !ok {
				return errors.New("instante inválido")
			}
			*puntero = valor
		default:
			return errors.New("destino no soportado")
		}
	}
	return nil
}

func TestConfirmacionAltaUsaContrato12x8YSesionCerrada(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	filaLocal := recibo
	filaLocal.ConfirmadaEn = filaLocal.ConfirmadaEn.In(
		time.FixedZone("zona-proceso", 2*60*60),
	)
	tx := nuevaTransaccionConfirmacionPrueba(filaLocal)
	pool := &iniciadorConfirmacionPrueba{transacciones: []pgx.Tx{tx}}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}
	parametros := parametrosConfirmacionPrueba()

	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametros,
	)
	if err != nil || obtenido != recibo {
		t.Fatalf("confirmación inesperada: %#v, %v", obtenido, err)
	}
	if tx.confirmaciones != 1 || tx.reversiones != 0 ||
		len(tx.argumentos) != 12 ||
		!strings.Contains(tx.consulta, funcionConfirmarAltaAtestadaV2) ||
		!strings.Contains(tx.configuracion, "'pg_catalog'") ||
		!strings.Contains(tx.configuracion, "'row_security', 'on'") ||
		!reflect.DeepEqual(tx.argumentos, argumentosConfirmacion(parametros)) {
		t.Fatalf("frontera SQL inesperada: %#v", tx)
	}
	if len(pool.opciones) != 1 ||
		pool.opciones[0].IsoLevel != pgx.Serializable ||
		pool.opciones[0].AccessMode != pgx.ReadWrite {
		t.Fatalf("aislamiento inesperado: %#v", pool.opciones)
	}
}

func TestConfirmacionAltaReconciliaRespuestaPerdidaTrasCommit(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	cancelado, cancelar := context.WithCancel(context.Background())
	txCommit := nuevaTransaccionConfirmacionPrueba(recibo)
	txCommit.errConfirmar = context.DeadlineExceeded
	txCommit.alConfirmar = cancelar
	txReconciliacion := nuevaTransaccionConfirmacionPrueba(recibo)
	pool := &iniciadorConfirmacionPrueba{
		transacciones: []pgx.Tx{txCommit, txReconciliacion},
	}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}
	parametros := parametrosConfirmacionPrueba()

	obtenido, err := adaptador.confirmarParametros(
		cancelado, expediente, parametros,
	)
	if err != nil || obtenido != recibo {
		t.Fatalf("no reconcilió el COMMIT: %#v, %v", obtenido, err)
	}
	if txCommit.confirmaciones != 1 || txReconciliacion.confirmaciones != 1 ||
		!reflect.DeepEqual(txCommit.argumentos, txReconciliacion.argumentos) ||
		!reflect.DeepEqual(
			txCommit.argumentos,
			argumentosConfirmacion(parametros),
		) {
		t.Fatalf("la reconciliación cambió el intento: %#v / %#v",
			txCommit.argumentos, txReconciliacion.argumentos)
	}
}

func TestConfirmacionAltaExponeIndeterminadoSiFallaSegundaEjecucion(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	txCommit := nuevaTransaccionConfirmacionPrueba(recibo)
	txCommit.errConfirmar = errors.New("respuesta COMMIT perdida")
	txReconciliacion := &transaccionConfirmacionPrueba{
		errConsulta: errors.New("base inaccesible"),
	}
	pool := &iniciadorConfirmacionPrueba{
		transacciones: []pgx.Tx{txCommit, txReconciliacion},
	}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if obtenido != (ports.ReciboAlta{}) ||
		!errors.Is(err, ports.ErrResultadoAltaIndeterminado) ||
		pool.inicios != 2 {
		t.Fatalf("resultado no cerrado: %#v, %v, intentos=%d",
			obtenido, err, pool.inicios)
	}
}

func TestConfirmacionAltaNoConfirmaReciboManipulado(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	recibo.ReciboHuellaSHA256 = strings.Repeat("f", 64)
	tx := nuevaTransaccionConfirmacionPrueba(recibo)
	adaptador := &TransaccionAltasPostgreSQL{
		pool: &iniciadorConfirmacionPrueba{transacciones: []pgx.Tx{tx}},
	}

	_, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if !errors.Is(err, ports.ErrResultadoAltaNoConfiable) ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf("recibo manipulado no falló cerrado: %v, %#v", err, tx)
	}
}

func TestConfirmacionAltaRechazaCardinalidadNulosYFormasNoCanonicas(
	t *testing.T,
) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	filaValida := filaReciboConfirmacionPrueba(recibo)
	casos := map[string][][]any{
		"cero filas": nil,
		"segunda fila": {
			append([]any(nil), filaValida...),
			append([]any(nil), filaValida...),
		},
		"NULL": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible, "1",
				recibo.ReciboRef, nil, recibo.EventoRef,
				recibo.ConfirmadaEn, recibo.ReciboHuellaSHA256,
			},
		},
		"version cero": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible, "0",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn, recibo.ReciboHuellaSHA256,
			},
		},
		"version con cero inicial": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible, "01",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn, recibo.ReciboHuellaSHA256,
			},
		},
		"version fuera de rango": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible,
				"18446744073709551616",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn, recibo.ReciboHuellaSHA256,
			},
		},
		"huella mayuscula": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible, "1",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn,
				strings.ToUpper(recibo.ReciboHuellaSHA256),
			},
		},
		"instante no canonico": {
			{
				recibo.ExpedienteRef, recibo.NumeroVisible, "1",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn.Add(time.Nanosecond),
				recibo.ReciboHuellaSHA256,
			},
		},
		"referencia invalida": {
			{
				"", recibo.NumeroVisible, "1",
				recibo.ReciboRef, recibo.AuditoriaRef, recibo.EventoRef,
				recibo.ConfirmadaEn, recibo.ReciboHuellaSHA256,
			},
		},
	}
	for nombre, filas := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &transaccionConfirmacionPrueba{
				filas: &filasConfirmacionPrueba{valores: filas},
			}
			adaptador := &TransaccionAltasPostgreSQL{
				pool: &iniciadorConfirmacionPrueba{
					transacciones: []pgx.Tx{tx},
				},
			}
			_, err := adaptador.confirmarParametros(
				context.Background(),
				expediente,
				parametrosConfirmacionPrueba(),
			)
			if !errors.Is(err, ports.ErrResultadoAltaNoConfiable) ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf("fila no fiable aceptada: %v, %#v", err, tx)
			}
		})
	}
}

func TestConfirmacionAltaNoReintentaErroresDeterminados(
	t *testing.T,
) {
	expediente := expedienteConfirmacionPrueba(t)
	casos := map[string]struct {
		err      error
		esperado error
	}{
		"23505": {
			err:      &pgconn.PgError{Code: "23505", Message: "secreto"},
			esperado: ports.ErrPersistenciaNoDisponible,
		},
		"42501": {
			err:      &pgconn.PgError{Code: "42501", Message: "secreto"},
			esperado: ports.ErrAutorizacionDenegada,
		},
		"55P03": {
			err:      &pgconn.PgError{Code: "55P03", Message: "secreto"},
			esperado: ports.ErrPersistenciaNoDisponible,
		},
		"57014": {
			err:      &pgconn.PgError{Code: "57014", Message: "secreto"},
			esperado: ports.ErrPersistenciaNoDisponible,
		},
		"forma": {
			err:      &pgconn.PgError{Code: "22023", Message: "secreto"},
			esperado: ports.ErrPersistenciaNoDisponible,
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &transaccionConfirmacionPrueba{errConsulta: caso.err}
			pool := &iniciadorConfirmacionPrueba{
				transacciones: []pgx.Tx{tx},
			}
			adaptador := &TransaccionAltasPostgreSQL{pool: pool}
			_, err := adaptador.confirmarParametros(
				context.Background(),
				expediente,
				parametrosConfirmacionPrueba(),
			)
			if !errors.Is(err, caso.esperado) || pool.inicios != 1 ||
				tx.confirmaciones != 0 || tx.reversiones != 1 ||
				strings.Contains(err.Error(), "secreto") {
				t.Fatalf("error determinado reinterpretado: %v, %#v", err, tx)
			}
		})
	}
}

func TestConfirmacionAltaReintentaConsultasConMismosDoceParametros(
	t *testing.T,
) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	txSerializacion := &transaccionConfirmacionPrueba{
		errConsulta: &pgconn.PgError{Code: "40001"},
	}
	txBloqueo := &transaccionConfirmacionPrueba{
		errConsulta: &pgconn.PgError{Code: "40P01"},
	}
	txFinal := nuevaTransaccionConfirmacionPrueba(recibo)
	pool := &iniciadorConfirmacionPrueba{transacciones: []pgx.Tx{
		txSerializacion, txBloqueo, txFinal,
	}}
	parametros := parametrosConfirmacionPrueba()
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametros,
	)
	esperados := argumentosConfirmacion(parametros)
	if err != nil || obtenido != recibo || pool.inicios != 3 ||
		!reflect.DeepEqual(txSerializacion.argumentos, esperados) ||
		!reflect.DeepEqual(txBloqueo.argumentos, esperados) ||
		!reflect.DeepEqual(txFinal.argumentos, esperados) ||
		txSerializacion.reversiones != 1 || txBloqueo.reversiones != 1 {
		t.Fatalf("reintento alteró el lote: %#v / %#v / %#v / %v",
			txSerializacion, txBloqueo, txFinal, err)
	}
}

func TestConfirmacionAltaReintentaSoloConflictosDeterminados(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	serializacion := &pgconn.PgError{Code: "40001"}
	bloqueo := &pgconn.PgError{Code: "40P01"}
	tx := nuevaTransaccionConfirmacionPrueba(recibo)
	pool := &iniciadorConfirmacionPrueba{
		errores:       []error{serializacion, bloqueo, nil},
		transacciones: []pgx.Tx{nil, nil, tx},
	}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if err != nil || obtenido != recibo || pool.inicios != 3 {
		t.Fatalf("presupuesto de reintento incorrecto: %#v, %v, %d",
			obtenido, err, pool.inicios)
	}

	for nombre, errCommit := range map[string]error{
		"directo":  pgx.ErrTxCommitRollback,
		"envuelto": fmt.Errorf("commit: %w", pgx.ErrTxCommitRollback),
	} {
		t.Run(nombre, func(t *testing.T) {
			txRollback := nuevaTransaccionConfirmacionPrueba(recibo)
			txRollback.errConfirmar = errCommit
			poolRollback := &iniciadorConfirmacionPrueba{
				transacciones: []pgx.Tx{txRollback},
			}
			adaptador.pool = poolRollback
			_, err := adaptador.confirmarParametros(
				context.Background(), expediente,
				parametrosConfirmacionPrueba(),
			)
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
				poolRollback.inicios != 1 {
				t.Fatalf("COMMIT determinado se reconcilió: %v, %d",
					err, poolRollback.inicios)
			}
		})
	}
}

func TestConfirmacionAltaNoReconciliaRespuestaPostgreSQLDeterminadaEnCommit(
	t *testing.T,
) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	tx := nuevaTransaccionConfirmacionPrueba(recibo)
	tx.errConfirmar = &pgconn.PgError{Code: "23505", Message: "secreto"}
	pool := &iniciadorConfirmacionPrueba{transacciones: []pgx.Tx{tx}}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	_, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		pool.inicios != 1 || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error PostgreSQL de COMMIT se reconcilió: %v, %d",
			err, pool.inicios)
	}
}

func TestConfirmacionAltaNoReconciliaCommitNoEnviado(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	tx := nuevaTransaccionConfirmacionPrueba(recibo)
	tx.errConfirmar = fmt.Errorf(
		"transporte: %w",
		errorReintentoSeguroPrueba{},
	)
	pool := &iniciadorConfirmacionPrueba{transacciones: []pgx.Tx{tx}}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	_, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		pool.inicios != 1 {
		t.Fatalf("COMMIT no enviado se reconcilió: %v, %d", err, pool.inicios)
	}
}

func TestConfirmacionAltaReconciliaResolucionPostgreSQLDesconocida(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	txCommit := nuevaTransaccionConfirmacionPrueba(recibo)
	txCommit.errConfirmar = &pgconn.PgError{Code: "08007"}
	txReconciliacion := nuevaTransaccionConfirmacionPrueba(recibo)
	pool := &iniciadorConfirmacionPrueba{
		transacciones: []pgx.Tx{txCommit, txReconciliacion},
	}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}

	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if err != nil || obtenido != recibo || pool.inicios != 2 {
		t.Fatalf("08007 no reconciliado: %#v, %v, %d",
			obtenido, err, pool.inicios)
	}
}

func TestConfirmacionAltaFallaCerradaSiReconciliacionDivergeOEsAmbigua(
	t *testing.T,
) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	divergente := recibo
	divergente.AuditoriaRef = "auditoria:alta-o2-06-divergente"
	var err error
	divergente.ReciboHuellaSHA256, err =
		ports.CalcularHuellaReciboAlta(divergente)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]pgx.Tx{
		"recibo divergente": nuevaTransaccionConfirmacionPrueba(divergente),
		"segundo commit ambiguo": func() pgx.Tx {
			tx := nuevaTransaccionConfirmacionPrueba(recibo)
			tx.errConfirmar = io.EOF
			return tx
		}(),
	}
	for nombre, segunda := range casos {
		t.Run(nombre, func(t *testing.T) {
			primera := nuevaTransaccionConfirmacionPrueba(recibo)
			primera.errConfirmar = io.EOF
			pool := &iniciadorConfirmacionPrueba{
				transacciones: []pgx.Tx{primera, segunda},
			}
			adaptador := &TransaccionAltasPostgreSQL{pool: pool}
			obtenido, err := adaptador.confirmarParametros(
				context.Background(), expediente,
				parametrosConfirmacionPrueba(),
			)
			if obtenido != (ports.ReciboAlta{}) ||
				!errors.Is(err, ports.ErrResultadoAltaIndeterminado) ||
				pool.inicios != 2 {
				t.Fatalf("reconciliación no cerró: %#v, %v", obtenido, err)
			}
		})
	}
}

func TestConfirmacionAltaDistingueTimeoutPrevioYPosteriorAlCommit(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	cancelado, cancelar := context.WithCancel(context.Background())
	txPrevio := &transaccionConfirmacionPrueba{
		errConsulta: context.DeadlineExceeded,
		alConsultar: cancelar,
	}
	adaptador := &TransaccionAltasPostgreSQL{
		pool: &iniciadorConfirmacionPrueba{
			transacciones: []pgx.Tx{txPrevio},
		},
	}
	_, err := adaptador.confirmarParametros(
		cancelado, expediente, parametrosConfirmacionPrueba(),
	)
	if !errors.Is(err, context.Canceled) || txPrevio.confirmaciones != 0 {
		t.Fatalf("timeout previo ambiguo: %v, %#v", err, txPrevio)
	}

	recibo := reciboConfirmacionPrueba(t, expediente)
	txPosterior := nuevaTransaccionConfirmacionPrueba(recibo)
	txPosterior.errConfirmar = context.DeadlineExceeded
	txReconciliacion := nuevaTransaccionConfirmacionPrueba(recibo)
	adaptador.pool = &iniciadorConfirmacionPrueba{
		transacciones: []pgx.Tx{txPosterior, txReconciliacion},
	}
	obtenido, err := adaptador.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if err != nil || obtenido != recibo {
		t.Fatalf("timeout posterior no reconciliado: %#v, %v", obtenido, err)
	}
}

func TestConfirmacionAltaReplayConcurrenteYTrasReinicio(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	recibo := reciboConfirmacionPrueba(t, expediente)
	pool := &iniciadorConfirmacionPrueba{fabrica: func() pgx.Tx {
		return nuevaTransaccionConfirmacionPrueba(recibo)
	}}
	adaptador := &TransaccionAltasPostgreSQL{pool: pool}
	const concurrencia = 24
	errores := make(chan error, concurrencia)
	for indice := 0; indice < concurrencia; indice++ {
		go func() {
			obtenido, err := adaptador.confirmarParametros(
				context.Background(),
				expediente,
				parametrosConfirmacionPrueba(),
			)
			if err == nil && obtenido != recibo {
				err = errors.New("recibo divergente")
			}
			errores <- err
		}()
	}
	for indice := 0; indice < concurrencia; indice++ {
		if err := <-errores; err != nil {
			t.Fatal(err)
		}
	}
	if pool.inicios != concurrencia {
		t.Fatalf("replays perdidos: %d", pool.inicios)
	}

	reiniciado := &TransaccionAltasPostgreSQL{
		pool: &iniciadorConfirmacionPrueba{fabrica: func() pgx.Tx {
			return nuevaTransaccionConfirmacionPrueba(recibo)
		}},
	}
	obtenido, err := reiniciado.confirmarParametros(
		context.Background(), expediente, parametrosConfirmacionPrueba(),
	)
	if err != nil || obtenido != recibo {
		t.Fatalf("replay tras reinicio divergente: %#v, %v", obtenido, err)
	}
}

func nuevaTransaccionConfirmacionPrueba(
	recibo ports.ReciboAlta,
) *transaccionConfirmacionPrueba {
	return &transaccionConfirmacionPrueba{
		filas: &filasConfirmacionPrueba{
			valores: [][]any{filaReciboConfirmacionPrueba(recibo)},
		},
	}
}

func filaReciboConfirmacionPrueba(recibo ports.ReciboAlta) []any {
	return []any{
		recibo.ExpedienteRef,
		recibo.NumeroVisible,
		strconv.FormatUint(recibo.Version, 10),
		recibo.ReciboRef,
		recibo.AuditoriaRef,
		recibo.EventoRef,
		recibo.ConfirmadaEn,
		recibo.ReciboHuellaSHA256,
	}
}

func reciboConfirmacionPrueba(
	t *testing.T,
	expediente domain.Expediente,
) ports.ReciboAlta {
	t.Helper()
	recibo := ports.ReciboAlta{
		ExpedienteRef: expediente.Referencia,
		NumeroVisible: expediente.NumeroVisible,
		Version:       expediente.Version,
		ReciboRef:     expediente.Actuaciones[0].ReciboRef,
		AuditoriaRef:  "auditoria:alta-o2-06-001",
		EventoRef:     "evento:alta-o2-06-001",
		ConfirmadaEn:  expediente.ActualizadoEn.Add(time.Microsecond),
	}
	var err error
	recibo.ReciboHuellaSHA256, err = ports.CalcularHuellaReciboAlta(recibo)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func expedienteConfirmacionPrueba(t *testing.T) domain.Expediente {
	t.Helper()
	instante := time.Date(2026, 7, 23, 9, 15, 0, 0, time.UTC)
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente:ct-o2-06-0001",
		OrganizacionRef: "organizacion:diputacion-granada",
		NumeroVisible:   "2026/CT-O2-06-0001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:contratacion-temporal-general",
			Version:       1,
			HuellaSHA256:  strings.Repeat("a", 64),
		},
		FaseInicial: "recepcion_solicitud",
		Solicitud: domain.SolicitudCentro{
			CentroRef:     "centro:residencia-rodriguez-penalva",
			ContactoRef:   "persona:responsable-centro-001",
			CategoriaRef:  "categoria:auxiliar-enfermeria",
			GrupoSubgrupo: "C2",
			MotivoClave:   "sustitucion.incapacidad_temporal",
			Detalle:       "Sustitución temporal para mantener la atención.",
			Periodo: domain.PeriodoPrevisto{
				Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				Fin:    time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
			},
			DocumentosAdjuntos: []string{"documento:informe-001"},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave:   "solicitud.registrada",
			ActorRef:      "persona:empleada-publica-001",
			UnidadRef:     "unidad:recursos-humanos",
			ReciboRef:     "recibo:alta-o2-06-001",
			RealizadaEn:   instante,
			FaseDestino:   "recepcion_solicitud",
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func parametrosConfirmacionPrueba() parametrosConfirmacionAlta {
	return parametrosConfirmacionAlta{
		capacidad:      []byte("capacidad"),
		decision:       []byte("decision"),
		motivo:         []byte("motivo"),
		contexto:       []byte("contexto"),
		personaVersion: "7",
		perfilVersion:  "11",
		payload:        []byte("payload"),
		cose:           []byte("cose"),
		evidencia:      []byte("evidencia"),
		spki:           []byte("spki"),
		alta:           []byte("alta"),
		sellos:         []byte("sellos"),
	}
}

func argumentosConfirmacion(p parametrosConfirmacionAlta) []any {
	return []any{
		p.capacidad, p.decision, p.motivo, p.contexto,
		p.personaVersion, p.perfilVersion, p.payload, p.cose,
		p.evidencia, p.spki, p.alta, p.sellos,
	}
}

func clonarArgumentosConfirmacion(argumentos []any) []any {
	copia := make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if contenido, ok := argumento.([]byte); ok {
			copia[indice] = append([]byte(nil), contenido...)
			continue
		}
		copia[indice] = argumento
	}
	return copia
}
