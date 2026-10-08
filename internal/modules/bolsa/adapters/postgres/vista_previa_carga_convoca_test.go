package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestLeerAcuseVistaPreviaCargaConvocaExigeUTCExacto(t *testing.T) {
	const valido = `{"decision_ref":"decision:1","acta_ref":"acta:1","huella_contexto":"abc","auditoria_ref":"aud_v3_0123456789abcdef0123456789abcdef","consumida_en":"2026-10-08T10:11:12.123456Z"}`
	acuse, err := leerAcuseVistaPreviaCargaConvoca([]byte(valido))
	if err != nil || acuse.ConsumidaEn.Location() != time.UTC || acuse.ConsumidaEn.Nanosecond() != 123456000 {
		t.Fatalf("acuse UTC microsegundo: %+v, %v", acuse, err)
	}
	for _, caso := range []string{
		`{"decision_ref":"decision:1","acta_ref":"acta:1","huella_contexto":"abc","auditoria_ref":"aud_v3_0123456789abcdef0123456789abcdef","consumida_en":"2026-10-08T10:11:12.123456+00:00"}`,
		`{"decision_ref":"decision:1","acta_ref":"acta:1","huella_contexto":"abc","auditoria_ref":"aud_v3_0123456789abcdef0123456789abcdef","consumida_en":"2026-10-08T10:11:12.1234567Z"}`,
		`{"decision_ref":"decision:1","acta_ref":"acta:1","huella_contexto":"abc","auditoria_ref":"aud_v3_0123456789abcdef0123456789abcdef","consumida_en":"2026-10-08T10:11:12.123456Z","filas":[]}`,
		valido + ` {}`,
	} {
		if _, err := leerAcuseVistaPreviaCargaConvoca([]byte(caso)); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) {
			t.Fatalf("acuse inesperado aceptado: %v", err)
		}
	}
}

func TestErrorConsumoVistaPreviaCargaConvocaCierraFallos(t *testing.T) {
	if err := errorConsumoVistaPreviaCargaConvoca(context.Background(), &pgconn.PgError{Code: "42501"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("42501: %v", err)
	}
	for _, errEntrada := range []error{&pgconn.PgError{Code: "40001"}, &pgconn.PgError{Code: "22023"}, errors.New("conexion cerrada")} {
		if err := errorConsumoVistaPreviaCargaConvoca(context.Background(), errEntrada); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) {
			t.Fatalf("fallo sin denegacion: %v", err)
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := errorConsumoVistaPreviaCargaConvoca(ctx, &pgconn.PgError{Code: "42501"}); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) {
		t.Fatalf("cancelacion debe prevalecer: %v", err)
	}
}

func TestConsumidorVistaPreviaCargaConvocaSinConexion(t *testing.T) {
	if _, err := NuevoConsumidorVistaPreviaCargaConvocaPostgreSQL(nil); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) {
		t.Fatalf("constructor sin pool: %v", err)
	}
	var consumidor *ConsumidorVistaPreviaCargaConvocaPostgreSQL
	if _, err := consumidor.ConsumirVistaPreviaCargaConvoca(context.Background(), ports.OrdenVistaPreviaCargaConvoca{},
		puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrVistaPreviaCargaConvocaNoDisponible) {
		t.Fatalf("consumidor sin pool: %v", err)
	}
}
