package postgres

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type txLimitesCargaConvocaPrueba struct {
	pgx.Tx
	ajustes []string
	err     error
}

func (t *txLimitesCargaConvocaPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.ajustes = append(t.ajustes, sql)
	return pgconn.CommandTag{}, t.err
}

func TestCargaConvocaFijaLimitesAntesDelEfecto(t *testing.T) {
	tx := &txLimitesCargaConvocaPrueba{}
	if err := prepararTransaccionCargaConvoca(context.Background(), tx); err != nil || len(tx.ajustes) != 4 ||
		tx.ajustes[0] != "SET LOCAL statement_timeout = '15s'" || tx.ajustes[1] != "SET LOCAL idle_in_transaction_session_timeout = '20s'" ||
		tx.ajustes[2] != "SET LOCAL lock_timeout = '2s'" || tx.ajustes[3] != "SET LOCAL timezone = 'UTC'" {
		t.Fatalf("límites de transacción incompletos: %v %v", tx.ajustes, err)
	}
	tx.err = &pgconn.PgError{Code: "42501", Message: "detalle personal sintético"}
	tx.ajustes = nil
	if err := prepararTransaccionCargaConvoca(context.Background(), tx); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) || len(tx.ajustes) != 1 {
		t.Fatalf("fallo al fijar límites no cerró la transacción: %v %v", tx.ajustes, err)
	}
}

func TestABICargaConvocaB95ColocaContextoAntesDelMaterialV3(t *testing.T) {
	constitucion := make([]any, 14)
	for i := range constitucion {
		constitucion[i] = i + 1
	}
	contexto := []byte(`{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{}}`)
	args, err := argumentosCargaConvocaB95([]byte(`{}`), []byte(`[]`), []byte(`{}`), constitucion, []byte(`[]`), contexto, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{})
	if err != nil || len(args) != 29 || !bytes.Equal(args[18].([]byte), contexto) || args[3] != 1 || args[16] != 14 ||
		!strings.Contains(consultaConfirmarCargaConvocaB95, "confirmar_carga_convoca_v2") ||
		!strings.Contains(consultaConfirmarCargaConvocaB95, "$18::jsonb, $19::bytea, $20::bytea") ||
		!strings.Contains(consultaConfirmarCargaConvocaB95, "$29::bytea") {
		t.Fatalf("ABI B95 divergente: argumentos=%d error=%v", len(args), err)
	}
}

func TestConstituirCargaConvocaAutorizadaFallaCerradaSinDependencias(t *testing.T) {
	var r *RepositorioCargaConvocaPostgreSQL
	if _, err := r.ConfirmarCargaConvocaAutorizada(context.Background(), importacion.LoteValidado{}, ports.Constitucion{}, nil, ports.OriginalProtegidoCargaConvoca{}, nil, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("repositorio nulo: %v", err)
	}
	r = &RepositorioCargaConvocaPostgreSQL{}
	if _, err := r.ConfirmarCargaConvocaAutorizada(context.Background(), importacion.LoteValidado{}, ports.Constitucion{}, nil, ports.OriginalProtegidoCargaConvoca{}, nil, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("sin pool ni material: %v", err)
	}
}

func TestErrorConstitucionCargaConvocaSeparaDenegacion(t *testing.T) {
	ctx := context.Background()
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "42501"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("42501: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "VA172"}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("VA172 debe ser fallo técnico, no denegación: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "23505"}); !errors.Is(err, ports.ErrConstitucionBolsaEnConflicto) {
		t.Fatalf("23505: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, errors.New("red")); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("otro: %v", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := errorConstitucionCargaConvoca(cancelado, &pgconn.PgError{Code: "42501"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelado: %v", err)
	}
}
