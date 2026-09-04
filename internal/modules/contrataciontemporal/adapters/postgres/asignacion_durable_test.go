package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorMaterialAsignacionPrueba struct {
	err error
}

func TestDecodificarReciboAsignacionPostgreSQLNormalizaInstanteUTC(
	t *testing.T,
) {
	t.Parallel()
	recibo, err := decodificarReciboConfirmacionAsignacion(
		`{"confirmada_en":"2026-09-04T13:50:52.123456789+00:00"}`,
	)
	if err != nil || !domain.InstanteUTCCanonico(recibo.ConfirmadaEn) ||
		recibo.ConfirmadaEn.Location() != time.UTC ||
		recibo.ConfirmadaEn.Nanosecond() != 123456000 {
		t.Fatalf("instante de recibo PostgreSQL no normalizado: %#v, %v", recibo, err)
	}
}

func TestDecodificarTerminalAsignacionPostgreSQLNormalizaInstanteUTC(
	t *testing.T,
) {
	t.Parallel()
	terminal, err := decodificarTerminalAsignacionPostgreSQL(
		`{"recibo":{"confirmada_en":"2026-09-04T13:50:52.654321987+00:00"}}`,
	)
	if err != nil || !domain.InstanteUTCCanonico(terminal.Recibo.ConfirmadaEn) ||
		terminal.Recibo.ConfirmadaEn.Location() != time.UTC ||
		terminal.Recibo.ConfirmadaEn.Nanosecond() != 654321000 {
		t.Fatalf("instante de terminal PostgreSQL no normalizado: %#v, %v", terminal, err)
	}
}

func (p *proveedorMaterialAsignacionPrueba) ProveerMaterialConfirmacionAsignacion(
	context.Context,
	ports.OrdenConfirmarAsignacion,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
}

func TestAdaptadoresAsignacionPostgreSQLRechazanDependenciasNulas(t *testing.T) {
	iniciador := &iniciadorPreparacionPrueba{}
	proveedor := &proveedorMaterialAsignacionPrueba{}
	if _, err := nuevoConsultorAsignacionPostgreSQL(nil); !errors.Is(
		err, ports.ErrPersistenciaAsignacionNoDisponible,
	) {
		t.Fatalf("consultor aceptó pool nulo: %v", err)
	}
	if _, err := nuevaTransaccionAsignacionesPostgreSQL(nil, proveedor); !errors.Is(
		err, ports.ErrPersistenciaAsignacionNoDisponible,
	) {
		t.Fatalf("confirmador aceptó pool nulo: %v", err)
	}
	if _, err := nuevaTransaccionAsignacionesPostgreSQL(iniciador, nil); !errors.Is(
		err, ports.ErrPersistenciaAsignacionNoDisponible,
	) {
		t.Fatalf("confirmador aceptó proveedor nulo: %v", err)
	}
}

func TestConsultorAsignacionPostgreSQLAusenciaEsCerrada(t *testing.T) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	preparacion := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(preparacion)
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{err: pgx.ErrNoRows},
	}
	consultor, err := nuevoConsultorAsignacionPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, encontrado, err := consultor.ConsultarAsignacion(
		context.Background(), consulta,
	)
	if err != nil || encontrado || !estado.EsCero() || tx.reversiones != 1 {
		t.Fatalf(
			"ausencia no cerrada: encontrado=%v cero=%v rollback=%d err=%v",
			encontrado, estado.EsCero(), tx.reversiones, err,
		)
	}
}

func TestConfirmacionAsignacionPostgreSQLFallaAntesDeAbrirTransaccion(
	t *testing.T,
) {
	iniciador := &iniciadorPreparacionPrueba{}
	confirmador, err := nuevaTransaccionAsignacionesPostgreSQL(
		iniciador,
		&proveedorMaterialAsignacionPrueba{err: errors.New("privado")},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = confirmador.ConfirmarAsignacion(
		context.Background(), ports.OrdenConfirmarAsignacion{},
	)
	if !errors.Is(err, ports.ErrPersistenciaAsignacionNoDisponible) ||
		iniciador.inicios != 0 {
		t.Fatalf("fallo de proveedor abrió transacción o se filtró: %v", err)
	}
}
