package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
)

type iniciadorSecuenciaConsultaRRHHPrueba struct {
	transacciones []*transaccionConsultaRRHHPrueba
	llamadas      int
	errBegin      error
}

func (i *iniciadorSecuenciaConsultaRRHHPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
		return nil, errors.New("opciones fuera de contrato")
	}
	if i.llamadas == 1 && i.errBegin != nil {
		return nil, i.errBegin
	}
	return i.transacciones[i.llamadas-1], nil
}

func nuevaTransaccionSecuenciaConsultaRRHHPrueba() *transaccionConsultaRRHHPrueba {
	eventos := []string{}
	return &transaccionConsultaRRHHPrueba{
		fila: &filaConsultaRRHHPrueba{eventos: &eventos}, eventos: &eventos,
	}
}

func TestCuadroRRHHReintenta40001SinCambiarMaterialNiCursor(t *testing.T) {
	t.Parallel()
	for _, etapa := range []string{"begin", "scan", "commit"} {
		t.Run(etapa, func(t *testing.T) {
			t.Parallel()
			primera := nuevaTransaccionSecuenciaConsultaRRHHPrueba()
			segunda := nuevaTransaccionSecuenciaConsultaRRHHPrueba()
			iniciador := &iniciadorSecuenciaConsultaRRHHPrueba{
				transacciones: []*transaccionConsultaRRHHPrueba{primera, segunda},
			}
			carrera := fmt.Errorf("operación SQL: %w", &pgconn.PgError{Code: "40001"})
			var recibo, cursor string
			primera.fila.llenar = func(destinos []any) error {
				*destinos[0].(*string), *destinos[1].(*string) = "recibo_abortado", "cursor_abortado"
				if etapa == "scan" {
					return carrera
				}
				return nil
			}
			primera.errCommit = carrera
			if etapa == "begin" {
				iniciador.errBegin = carrera
			}
			segunda.fila.llenar = func(destinos []any) error {
				if etapa != "begin" && primera.reversiones != 1 {
					t.Fatal("nuevo intento antes de cerrar la transacción abortada")
				}
				*destinos[0].(*string), *destinos[1].(*string) = "recibo_confirmado", "cursor_confirmado"
				return nil
			}
			argumentos := []any{"organizacion", "unidad", "filtro", "cursor_entrada", []byte("material_original")}
			validaciones := 0
			resultado, err := ejecutarConsultaRRHHEnTransaccion(context.Background(), iniciador,
				consultaCuadroRRHHPostgreSQL, argumentos, []any{&recibo, &cursor},
				func() (string, error) {
					validaciones++
					return recibo + "/" + cursor, nil
				})
			if err != nil || resultado != "recibo_confirmado/cursor_confirmado" || iniciador.llamadas != 2 {
				t.Fatalf("resultado=%q err=%v intentos=%d", resultado, err, iniciador.llamadas)
			}
			if segunda.confirmaciones != 1 || segunda.reversiones != 1 ||
				segunda.consulta != consultaCuadroRRHHPostgreSQL || !reflect.DeepEqual(segunda.valores, argumentos) {
				t.Fatal("segundo intento alteró SQL/material o no cerró su transacción")
			}
			if etapa != "begin" && (!reflect.DeepEqual(primera.valores, segunda.valores) || primera.reversiones != 1) {
				t.Fatal("material/cursor cambiado entre intentos")
			}
			validacionesEsperadas := 1
			if etapa == "commit" {
				validacionesEsperadas = 2
			}
			if validaciones != validacionesEsperadas {
				t.Fatalf("validaciones=%d esperadas=%d", validaciones, validacionesEsperadas)
			}
		})
	}
}

func TestCuadroRRHHNoReintentaOtrosErroresNiValidacion(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre, etapa, consulta string
		causa, esperado         error
	}{
		{"permiso", "scan", consultaCuadroRRHHPostgreSQL, &pgconn.PgError{Code: "42501"}, ports.ErrConsultaRRHHNoObservable},
		{"deadlock", "scan", consultaCuadroRRHHPostgreSQL, &pgconn.PgError{Code: "40P01"}, ports.ErrConsultaRRHHNoDisponible},
		{"bloqueo", "scan", consultaCuadroRRHHPostgreSQL, &pgconn.PgError{Code: "55P03"}, ports.ErrConsultaRRHHNoDisponible},
		{"cancelacion_SQL", "scan", consultaCuadroRRHHPostgreSQL, &pgconn.PgError{Code: "57014"}, ports.ErrConsultaRRHHNoDisponible},
		{"confirmacion_ambigua", "commit", consultaCuadroRRHHPostgreSQL, errors.New("respuesta perdida"), ports.ErrConsultaRRHHNoDisponible},
		{"validacion_40001", "validar", consultaCuadroRRHHPostgreSQL, &pgconn.PgError{Code: "40001"}, nil},
		{"detalle_40001", "scan", consultaDetalleRRHHPostgreSQL, &pgconn.PgError{Code: "40001"}, ports.ErrConsultaRRHHNoDisponible},
		{"resumen_40001", "commit", consultaResumenSeguimientoRRHHPostgreSQL, &pgconn.PgError{Code: "40001"}, ports.ErrConsultaRRHHNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			tx := nuevaTransaccionSecuenciaConsultaRRHHPrueba()
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			var errValidar error
			switch caso.etapa {
			case "scan":
				tx.fila.err = caso.causa
			case "commit":
				tx.errCommit = caso.causa
			case "validar":
				errValidar = caso.causa
				caso.esperado = caso.causa
			}
			resultado, err := ejecutarConsultaRRHHEnTransaccion(context.Background(), iniciador,
				caso.consulta, nil, nil, func() (string, error) { return "validado", errValidar })
			if resultado != "" || !errors.Is(err, caso.esperado) || iniciador.llamadas != 1 || tx.reversiones != 1 {
				t.Fatalf("resultado=%q err=%v intentos=%d rollback=%d", resultado, err, iniciador.llamadas, tx.reversiones)
			}
		})
	}
}

func TestCuadroRRHHReintentoAcotadoYErrorOriginal(t *testing.T) {
	t.Parallel()
	iniciador := &iniciadorSecuenciaConsultaRRHHPrueba{}
	causa := &pgconn.PgError{Code: "40001", Message: "dato privado"}
	for range postgresqlcomun.IntentosMaximosCarreraSerializable {
		tx := nuevaTransaccionSecuenciaConsultaRRHHPrueba()
		tx.fila.err = causa
		iniciador.transacciones = append(iniciador.transacciones, tx)
	}
	resultado, err := ejecutarConsultaRRHHEnTransaccion(context.Background(), iniciador,
		consultaCuadroRRHHPostgreSQL, nil, nil, func() (string, error) {
			t.Fatal("resultado validado tras scan fallido")
			return "", nil
		})
	if resultado != "" || !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || !errors.Is(err, causa) ||
		err.Error() != "consulta RRHH: sql" || iniciador.llamadas != postgresqlcomun.IntentosMaximosCarreraSerializable {
		t.Fatalf("resultado=%q err=%v intentos=%d", resultado, err, iniciador.llamadas)
	}
	for _, tx := range iniciador.transacciones {
		if tx.reversiones != 1 || tx.confirmaciones != 0 {
			t.Fatal("intento fallido confirmó o dejó abierta su transacción")
		}
	}
}

func TestCuadroRRHHReintentoRespetaCancelacion(t *testing.T) {
	t.Parallel()
	for _, etapa := range []string{"scan", "rollback"} {
		t.Run(etapa, func(t *testing.T) {
			t.Parallel()
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			tx := nuevaTransaccionSecuenciaConsultaRRHHPrueba()
			tx.fila.err = &pgconn.PgError{Code: "40001"}
			if etapa == "rollback" {
				tx.alRevertir = cancelar
			} else {
				tx.fila.llenar = func([]any) error {
					cancelar()
					return tx.fila.err
				}
			}
			iniciador := &iniciadorConsultaRRHHPrueba{tx: tx}
			resultado, err := ejecutarConsultaRRHHEnTransaccion(ctx, iniciador, consultaCuadroRRHHPostgreSQL,
				nil, nil, func() (string, error) { return "", nil })
			if resultado != "" || !errors.Is(err, context.Canceled) || iniciador.llamadas != 1 || tx.reversiones != 1 {
				t.Fatalf("resultado=%q err=%v intentos=%d", resultado, err, iniciador.llamadas)
			}
		})
	}
}

func TestCuadroRRHHNoIniciaConContextoCanceladoOVencido(t *testing.T) {
	t.Parallel()
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	vencido, cerrar := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cerrar()
	for _, ctx := range []context.Context{cancelado, vencido} {
		iniciador := &iniciadorConsultaRRHHPrueba{}
		resultado, err := ejecutarConsultaRRHHEnTransaccion(ctx, iniciador, consultaCuadroRRHHPostgreSQL,
			nil, nil, func() (string, error) { return "", nil })
		if resultado != "" || !errors.Is(err, ctx.Err()) || iniciador.llamadas != 0 {
			t.Fatalf("resultado=%q err=%v intentos=%d", resultado, err, iniciador.llamadas)
		}
	}
}
